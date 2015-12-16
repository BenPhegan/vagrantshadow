package main

import (
	"encoding/json"
	"expvar"
	"flag"
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/go-fsnotify/fsnotify"
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var boxDownloads = expvar.NewInt("box_downloads")
var boxQueries = expvar.NewInt("box_queries")
var boxChecks = expvar.NewInt("box_checks")
var homepageVisits = expvar.NewInt("homepage_visits")
var boxStats = expvar.NewMap("box_stats")

func getBox(bh *BoxHandler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		user := vars["user"]
		boxName := vars["boxname"]

		boxQueries.Add(1)
		log.Println("Queried for " + user + "/" + boxName)
		box := bh.GetBox(user, boxName)

		jsonResponse, _ := json.Marshal(box)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonResponse)
	}
	return http.HandlerFunc(fn)
}

func downloadBox(bh *BoxHandler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		user := vars["user"]
		boxName := vars["boxname"]
		provider := vars["provider"]
		version := vars["version"]
		log.Println("Downloading " + user + "/" + boxName + "/" + version + "/" + provider)

		boxDownloads.Add(1)
		boxStats.Add(strings.Join([]string{user, "/", boxName, "/", provider, "/", version}, ""), 1)
		http.ServeFile(w, r, bh.GetBoxFileLocation(user, boxName, provider, version))
	}
	return http.HandlerFunc(fn)
}

func checkBox(bh *BoxHandler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		user := vars["user"]
		boxName := vars["boxname"]
		log.Println("Checking " + user + "/" + boxName)

		boxChecks.Add(1)
		if bh.BoxAvailable(user, boxName) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
	return http.HandlerFunc(fn)
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

func notFound(w http.ResponseWriter, r *http.Request) {
	log.Println("404 :", r.URL.Path, " ", r.Method)
	w.WriteHeader(http.StatusNotFound)
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
	log.Println("Using box regex:" + bh.BoxRegex())
	bh.Hostname = *hostname
	bh.Port = *port
	bh.PopulateBoxes(directories, port, hostname)
	home.BoxHandler = &bh
	home.TemplateString = home.GetTemplateString(*templatefile)

	setUpFileWatcher(directories, func() { bh.PopulateBoxes(directories, port, hostname) })

	m := mux.NewRouter()
	m.Handle("/{user}/{boxname}", getBox(&bh)).Methods("GET")
	m.Handle("/{user}/{boxname}", checkBox(&bh)).Methods("HEAD")
	m.Handle("/", showHomepage(&home)).Methods("GET")
	//Handling downloads that look like Vagrant Cloud
	//https://vagrantcloud.com/benphegan/boot2docker/version/2/provider/vmware_desktop.box
	m.Handle("/{user}/{boxname}/{version}/{provider}/{boxfile}", downloadBox(&bh)).Methods("GET")
	m.NotFoundHandler = http.HandlerFunc(notFound)
	http.Handle("/", m)

	log.Println("Listening on port: ", *port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
