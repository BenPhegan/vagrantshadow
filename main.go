package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"expvar"
	"flag"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var boxDownloads = expvar.NewInt("box_downloads")
var boxQueries = expvar.NewInt("box_queries")
var boxChecks = expvar.NewInt("box_checks")
var homepageVisits = expvar.NewInt("homepage_visits")
var boxStats = expvar.NewMap("box_stats")

func (bh BoxHandler) GetBox(w http.ResponseWriter, r *http.Request) {
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

func (bh BoxHandler) DownloadBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	boxName := vars["boxname"]
	log.Println("Downloading ", user, "/", boxName)

	boxDownloads.Add(1)
	boxStats.Add(strings.Join([]string{user, "/", boxName}, ""), 1)
	localBoxfile := bh.Boxes[user][boxName]
	http.ServeFile(w, r, localBoxfile.LocalBoxFile)
}

func (bh BoxHandler) CheckBox(w http.ResponseWriter, r *http.Request) {
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

func (bh BoxHandler) ShowHomepage(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("homepage").Parse(bh.TemplateString)
	if err != nil {
		w.Write([]byte("Could not parse provided template: " + err.Error()))
		return
	}
	t.Execute(w, bh)
}

func (bh BoxHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Println("404 :", r.URL.Path, " ", r.Method)
	w.WriteHeader(http.StatusNotFound)
}

func main() {

	directory := flag.String("d", "./", "Semicolon separated list of directories containing .box files")
	port := flag.Int("p", 8099, "Port to listen on.")
	hostname := flag.String("h", "localhost", "Hostname for static box content.")
	templatefile := flag.String("t", "", "Template file for the vagrantshadow homepage, if you dont like the default!")
	writeouttemplate := flag.Bool("w", false, "Write a template page to disk so you can modify")

	flag.Parse()

	if *writeouttemplate {
		//output a template homepage so people have something to play
		outputTemplateString("hometemplate.html")
	}

	directories := strings.Split(*directory, ";")
	absolutedirectories := []string{}
	for _, d := range directories {
		if !path.IsAbs(d) {
			wd, _ := os.Getwd()
			absolute := path.Clean(path.Join(wd, d))
			absolutedirectories = append(absolutedirectories, absolute)
		}
	}

	log.Println("Responding on host: ", *hostname)
	log.Println("Serving files from: ", *directory)
	boxfiles := getBoxList(absolutedirectories)
	boxdata := getBoxData(boxfiles)
	boxes := createBoxes(boxdata, *port, hostname)

	for namespace, boxinfo := range boxes {
		for boxname, _ := range boxinfo {
			log.Println(strings.Join([]string{"Found: ", namespace, "/", boxname}, ""))
		}
	}

	bh := BoxHandler{}
	bh.Boxes = boxes
	bh.Hostname = *hostname
	bh.Directories = absolutedirectories
	bh.Port = *port
	bh.TemplateString = getTemplateString(*templatefile)

	m := mux.NewRouter()
	m.HandleFunc("/{user}/{boxname}", bh.GetBox).Methods("GET")
	m.HandleFunc("/{user}/{boxname}", bh.CheckBox).Methods("HEAD")
	m.HandleFunc("/", bh.ShowHomepage).Methods("GET")
	//Handling downloads that look like Vagrant Cloud
	//https://vagrantcloud.com/benphegan/boot2docker/version/2/provider/vmware_desktop.box
	m.HandleFunc("/{user}/{boxname}/{version}/provider/{boxfile}", bh.DownloadBox).Methods("GET")
	m.NotFoundHandler = http.HandlerFunc(bh.NotFound)
	http.Handle("/", m)

	log.Println("Listening on port: ", *port)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)
}

type BoxHandler struct {
	Boxes          map[string]map[string]Box
	Hostname       string
	Port           int
	Directories    []string
	TemplateString string
}

func getTemplateString(location string) string {
	if _, err := os.Stat(location); err == nil {
		log.Println("Found template file: " + location)
		templatetext, err := ioutil.ReadFile(location)
		if err != nil {
			log.Println("Could not load template: " + location)
			return getDefaultTemplateString()
		}
		template := string(templatetext)
		return template
	}
	return getDefaultTemplateString()
}

func outputTemplateString(location string) {
	if _, err := os.Stat(location); os.IsNotExist(err) {
		log.Println("Writing out default home template file: " + location)
		err := ioutil.WriteFile(location, []byte(getDefaultTemplateString()), 0644)
		if err != nil {
			log.Println("Failed to write default template to: " + location + " - " + err.Error())
		}
	} else {
		log.Println("Default template exists on disk already")
	}
}

func getDefaultTemplateString() string {
	return `<html>
		<h1>vagrantshadow</h1>
		Welcome to vagrantshadow.
		<h2>Configuration</h2>
		<p>To use vagrantshadow with Vagrant:</p>
		<ul>
			<li><strong>Mac/Unix</strong> - <tt>export VAGRANT_SERVER_URL=http://{{ .Hostname }}:{{ .Port }}</tt></li>
			<li><strong>Windows</strong> - <tt>set VAGRANT_SERVER_URL=http://{{ .Hostname }}:{{ .Port }}</tt></li>
		</ul>
		<br>
		<h2>Available Boxes</h2>
		{{ range $index, $element := .Boxes }}
			{{ range $key, $value := $element }}
				{{ $value.Name }} <br>
			{{end }}
		{{ end }}
		<h2>Statistics</h2>
		<a HREF="http://{{ .Hostname }}:{{ .Port }}/debug/vars">Debug Variables</a>
	</html>`
}

//Creates the data structure used to provide box data to Vagrant
func createBoxes(sb []SimpleBox, port int, hostname *string) map[string]map[string]Box {
	boxes := make(map[string]map[string]Box)
	for _, b := range sb {
		box := Box{}
		box.Name = b.Username + "/" + b.Boxname
		box.Username = b.Username
		box.Private = false
		box.LocalBoxFile = b.Location

		provider := Provider{}
		provider.Name = b.Provider
		provider.Hosted = "true"
		provider.DownloadUrl = "http://" + *hostname + ":" + strconv.Itoa(port) + "/" + b.Username + "/" + b.Boxname + "/1/provider/" + b.Provider + ".box"
		provider.Url = provider.DownloadUrl
		version := Version{}
		version.Status = "active"
		version.Version = "1.0"

		version.Providers = []Provider{provider}

		box.Versions = []Version{version}
		box.CurrentVersion = version
		boxes[b.Username] = make(map[string]Box)
		boxes[b.Username][b.Boxname] = box
	}
	return boxes
}

// getBoxList returns a list of .box files in the directories provided.
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
	for _, b := range boxfiles {
		nameparts := strings.Split(filepath.Base(b), "-VAGRANTSLASH-")
		if len(nameparts) == 2 {
			provider, _ := getProvider(b)
			newbox := SimpleBox{}
			newbox.Username = nameparts[0]
			newbox.Boxname = nameparts[1][0 : len(nameparts[1])-4]
			newbox.Location = b
			newbox.Provider = provider.Provider
			results = append(results, newbox)
		}
	}
	return results
}

// Gets the provider information from the Vagrant .box files
func getProvider(location string) (BoxMetadata, error) {

	log.Println("Checking: ", location)
	//boxes are either .tar, .tar.gz or .zip so we have to get a gzip stream and pass to tar, just tar or zip check

	// Check zip first...
	newbox, unzipError := getProviderFromZip(location)
	if unzipError != nil {
		log.Println("Not a zip file: " + location + " - " + unzipError.Error())
	} else {
		return newbox, nil
	}

	// Check tar and gzip
	f, openerr := os.Open(location)
	if openerr != nil {
		openerrmessage := "Could not open file: " + location + " - " + openerr.Error()
		log.Fatalln(openerrmessage)
		return BoxMetadata{}, errors.New(openerrmessage)
	}

	// Try gzip first, fall back to tar
	var tr *tar.Reader
	r, gziperr := gzip.NewReader(f)
	if gziperr != nil {
		gziperrmessage := "Could not get GZIP reader: " + location + " - " + gziperr.Error()
		log.Println(gziperrmessage)
		f.Seek(0, 0)
		tr = tar.NewReader(f)
	} else {
		tr = tar.NewReader(r)
	}

	// Iterate through the files in the archive.
	// Some boxes do not have a metadata.json, generally old virtualbox boxes
	// In this case, fall through to just checking them for an ovf and returning provider
	metadatafound := false
	ovf := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			if !metadatafound {
				log.Println("Could not find metadata.json in file: " + location)
				if ovf {
					log.Println("Found OVF, creating for VirtualBox: " + location)
					metadata := BoxMetadata{}
					metadata.Provider = "virtualbox"
					return metadata, nil
				}
			}
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		if filepath.Ext(hdr.Name) == ".ovf" {
			ovf = true
		}

		if hdr.Name == "metadata.json" {
			metadatafound = true
			metadata := BoxMetadata{}

			boxmetadata := BoxMetadata{}

			buf := new(bytes.Buffer)
			buf.ReadFrom(tr)

			json.Unmarshal(buf.Bytes(), &boxmetadata)
			metadata.Provider = boxmetadata.Provider
			return metadata, nil
		}

	}
	return BoxMetadata{}, errors.New("box: could not find metadata.json or box malformed")
}

// Gets the provider information fromt the Vagrant .box files using Zip
func getProviderFromZip(src string) (BoxMetadata, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return BoxMetadata{}, err
	}
	defer r.Close()

	for _, f := range r.File {

		if f.Name == "metadata.json" {
			log.Println("ZIP!!!")
		}

		rc, err := f.Open()
		if err != nil {
			return BoxMetadata{}, err
		}
		defer rc.Close()

		metadata := BoxMetadata{}

		boxmetadata := BoxMetadata{}

		buf := new(bytes.Buffer)
		buf.ReadFrom(rc)

		json.Unmarshal(buf.Bytes(), &boxmetadata)
		metadata.Provider = boxmetadata.Provider
		return metadata, nil

	}

	return BoxMetadata{}, nil
}

type BoxMetadata struct {
	Provider string `json:"provider"`
}

type SimpleBox struct {
	Username string
	Boxname  string
	Location string
	Provider string
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
	CurrentVersion      Version   `json:"current_version"`
	Versions            []Version `json:"versions"`
	LocalBoxFile        string    `json:"-"`
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
	Name        string `json:"name"`
	Hosted      string `json:"hosted"`
	HostedToken string `json:"hosted_token"`
	OriginalUrl string `json:"original_url"`
	UploadUrl   string `json:"upload_url"`
	Created     string `json:"created_at"`
	Updated     string `json:"updated_at"`
	DownloadUrl string `json:"download_url"`
	Url         string `json:"url"`
}
