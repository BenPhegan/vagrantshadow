package main

import (
	"encoding/json"
	"expvar"
	"flag"
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/go-fsnotify/fsnotify"
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/mcuadros/go-version"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var boxDownloads = expvar.NewInt("box_downloads")
var boxQueries = expvar.NewInt("box_queries")
var boxChecks = expvar.NewInt("box_checks")
var homepageVisits = expvar.NewInt("homepage_visits")
var boxStats = expvar.NewMap("box_stats")

func (bh *BoxHandler) GetBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	boxName := vars["boxname"]

	boxQueries.Add(1)
	log.Println(strings.Join([]string{"Queried for ", user, "/", boxName}, ""))
	box := bh.Boxes[user][boxName]

	jsonResponse, _ := json.Marshal(box)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonResponse)
}

func (bh *BoxHandler) DownloadBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	for _, v := range vars {
		log.Println(v)
	}
	user := vars["user"]
	boxName := vars["boxname"]
	log.Println("Downloading ", user, "/", boxName)

	boxDownloads.Add(1)
	boxStats.Add(strings.Join([]string{user, "/", boxName}, ""), 1)
	provider := bh.Boxes[user][boxName].CurrentVersion.Providers[0]
	http.ServeFile(w, r, provider.LocalBoxFile)
}

func (bh *BoxHandler) CheckBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	boxName := vars["boxname"]
	log.Println(strings.Join([]string{"Checking ", user, "/", boxName}, ""))

	boxChecks.Add(1)
	localBoxfile := bh.Boxes[user][boxName]
	if localBoxfile.Username != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func showHomepage(ht *HomePageTemplate) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("homepage").Parse(ht.TemplateString)
		if err != nil {
			w.Write([]byte("Could not parse provided template: " + err.Error()))
			return
		}
		err = t.Execute(w, ht.BoxHandler)
		if err != nil {
			log.Fatalln("Failed to execute homepage template: " + err.Error())
		}
	}
	return http.HandlerFunc(fn)
}

func (bh *BoxHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Println("404 :", r.URL.Path, " ", r.Method)
	w.WriteHeader(http.StatusNotFound)
}

func (bh *BoxHandler) PopulateBoxes(directories []string, port *int, hostname *string) {
	absolutedirectories := []string{}
	for _, d := range directories {
		if !path.IsAbs(d) {
			wd, _ := os.Getwd()
			absolute := path.Clean(path.Join(wd, d))
			absolutedirectories = append(absolutedirectories, absolute)
		}
	}

	boxfiles := getBoxList(absolutedirectories)
	boxdata := getBoxData(boxfiles)
	boxes := createBoxes(boxdata, *port, hostname)
	bh.Boxes = boxes

	for namespace, boxinfo := range boxes {
		for boxname, _ := range boxinfo {
			log.Println(strings.Join([]string{"Found: ", namespace, "/", boxname}, ""))
		}
	}
}

func setUpFileWatcher(directories []string, action func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Could not create file watcher, updates to file system will not be picked up.")
	}
	for _, d := range directories {
		log.Println("Setting directory watch on : " + d)
		watcher.Add(d)
	}
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				dirname := filepath.Dir(ev.Name)
				log.Println("Directory change detected: " + dirname)
				action()
			case err := <-watcher.Errors:
				log.Fatalln("error:", err)
			}
		}
	}()
}

func main() {

	//We expect files in the format:
	// owner-VAGRANTSLASH-boxname__provider__version.box

	directory := flag.String("d", "./", "Semicolon separated list of directories containing .box files")
	port := flag.Int("p", 8099, "Port to listen on.")
	hostname := flag.String("h", "localhost", "Hostname for static box content.")
	templatefile := flag.String("t", "", "Template file for the vagrantshadow homepage, if you dont like the default!")
	writeouttemplate := flag.Bool("w", false, "Write a template page to disk so you can modify")

	flag.Parse()

	home := HomePageTemplate{}
	if *writeouttemplate {
		//output a template homepage so people have something to play
		home.OutputTemplateString("hometemplate.html")
	}

	directories := strings.Split(*directory, ";")

	//Add a default of . so we dont always have to add a directory...
	if len(directories) == 0 {
		directories = append(directories, ".")
	}

	log.Println("Responding on host: ", *hostname)
	log.Println("Serving files from: ", *directory)

	bh := BoxHandler{}
	bh.Hostname = *hostname
	bh.Port = *port
	bh.PopulateBoxes(directories, port, hostname)
	home.BoxHandler = &bh
	home.TemplateString = home.GetTemplateString(*templatefile)

	setUpFileWatcher(directories, func() { bh.PopulateBoxes(directories, port, hostname) })

	m := mux.NewRouter()
	m.HandleFunc("/{user}/{boxname}", bh.GetBox).Methods("GET")
	m.HandleFunc("/{user}/{boxname}", bh.CheckBox).Methods("HEAD")
	m.Handle("/", showHomepage(&home)).Methods("GET")
	//Handling downloads that look like Vagrant Cloud
	//https://vagrantcloud.com/benphegan/boot2docker/version/2/provider/vmware_desktop.box
	m.HandleFunc("/{user}/{boxname}/{version}/{provider}/{boxfile}", bh.DownloadBox).Methods("GET")
	m.NotFoundHandler = http.HandlerFunc(bh.NotFound)
	http.Handle("/", m)

	log.Println("Listening on port: ", *port)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)
}

type BoxHandler struct {
	Boxes          map[string]map[string]Box
	Directories    []string
	TemplateString string
	Hostname       string
	Port           int
}

//Creates the data structure used to provide box data to Vagrant
func createBoxes(sb []SimpleBox, port int, hostname *string) map[string]map[string]Box {
	boxes := make(map[string]map[string]Box)
	for _, b := range sb {
		box := Box{}
		box.Name = b.Username + "/" + b.Boxname
		box.Username = b.Username
		box.Private = false

		provider := Provider{}
		provider.Name = b.Provider
		provider.Hosted = "true"
		provider.DownloadUrl = "http://" + *hostname + ":" + strconv.Itoa(port) + "/" + b.Username + "/" + b.Boxname + "/" + b.Version + "/" + b.Provider + "/" + b.Provider + ".box"
		provider.Url = provider.DownloadUrl
		provider.LocalBoxFile = b.Location

		if len(box.Versions) > 0 {
			for _, v := range box.Versions {
				if v.Version == b.Version {
					v.Providers = append(v.Providers, provider)
				}
			}
		} else {
			newversion := Version{}
			newversion.Status = "active"
			newversion.Version = b.Version
			newversion.Providers = []Provider{provider}
			box.Versions = []Version{newversion}
			if box.CurrentVersion == nil {
				box.CurrentVersion = &newversion
			} else {
				if version.Compare(box.CurrentVersion.Version, newversion.Version, "<") {
					box.CurrentVersion = &newversion
				}
			}
		}

		if boxes[b.Username] == nil {
			boxes[b.Username] = make(map[string]Box)
		}
		boxes[b.Username][b.Boxname] = box
	}

	return boxes
}

// getBoxList returns a list of .box files in the directories provided.
// Returns full path
func getBoxList(directories []string) []string {
	boxes := []string{}
	for _, d := range directories {
		directoryglob := path.Join(d, "*.box")
		files, _ := filepath.Glob(directoryglob)
		boxes = append(boxes, files...)
	}
	return boxes
}

//getBoxData returns an array of SimpleBox objects based on Vagrant box files
func getBoxData(boxfiles []string) []SimpleBox {
	results := []SimpleBox{}
	var myExp = regexp.MustCompile(`(?P<owner>\w*)-VAGRANTSLASH-(?P<boxname>[a-zA-Z0-9]*)__(?P<provider>[a-zA-Z0-9]*)__(?P<version>[a-zA-Z0-9\.-]*).box`)
	for _, b := range boxfiles {
		matches := myExp.FindStringSubmatch(filepath.Base(b))
		if matches == nil || len(matches) != 5 {
			log.Println("Could not match metadata from filename: " + filepath.Base(b))
			return results
		}
		newbox := SimpleBox{Username: matches[1], Boxname: matches[2], Location: b, Provider: matches[3], Version: matches[4]}
		results = append(results, newbox)
	}

	return results
}

type BoxMetadata struct {
	Provider string `json:"provider"`
}

type SimpleBox struct {
	Username string
	Boxname  string
	Location string
	Provider string
	Version  string
}

type Box struct {
	Created             string    `json:"created_at"`
	Updated             string    `json:"updated_at"`
	Tag                 string    `json:"tag"`
	Name                string    `json:"name"`
	ShortDescription    string    `json:"short_description"`
	DescriptionHtml     string    `json:"description_html"`
	DescriptionMarkdown string    `json:"description_markdown"`
	Username            string    `json:"username"`
	Private             bool      `json:"private"`
	CurrentVersion      *Version  `json:"current_version"`
	Versions            []Version `json:"versions"`
}

type Version struct {
	Version             string     `json:"version"`
	Status              string     `json:"status"`
	DescriptionHtml     string     `json:"description_html"`
	DescriptionMarkdown string     `json:"description_markdown"`
	Created             string     `json:"created_at"`
	Updated             string     `json:"updated_at"`
	Number              int        `json:"number"`
	Downloads           int        `json:"downloads"`
	ReleaseUrl          string     `json:"release_url"`
	RevokeUrl           string     `json:"revoke_url"`
	Providers           []Provider `json:"providers"`
}

type Provider struct {
	Name         string `json:"name"`
	Hosted       string `json:"hosted"`
	HostedToken  string `json:"hosted_token"`
	OriginalUrl  string `json:"original_url"`
	UploadUrl    string `json:"upload_url"`
	Created      string `json:"created_at"`
	Updated      string `json:"updated_at"`
	DownloadUrl  string `json:"download_url"`
	Url          string `json:"url"`
	LocalBoxFile string `json:"-"`
}
