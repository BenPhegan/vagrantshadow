package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func (bh BoxHandler) GetBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	boxName := vars["boxname"]
	log.Println("Queried for ", user, "/", boxName)
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

	localBoxfile := bh.Boxes[user][boxName]
	http.ServeFile(w, r, localBoxfile.LocalBoxFile)
}

func (bh BoxHandler) CheckBox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	boxName := vars["boxname"]
	log.Println("Checking ", user, "/", boxName)

	localBoxfile := bh.Boxes[user][boxName]
	if localBoxfile.Username != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
func (bh BoxHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Println("404 :", r.URL.Path, " ", r.Method)
	w.WriteHeader(http.StatusNotFound)
}

func main() {

	directory := flag.String("d", "./", "Base directory containing .box files")
	port := flag.Int("port", 8099, "Port to listen on.")
	hostname := flag.String("hostname", "localhost", "Hostname for static box content.")

	flag.Parse()

	absolute := *directory
	if !path.IsAbs(*directory) {
		wd, _ := os.Getwd()
		absolute = path.Clean(path.Join(wd, *directory))
	}

	log.Println("Responding on host: ", *hostname)
	log.Println("Serving files from: ", *directory)
	boxfiles := getBoxList(absolute)
	boxdata := getBoxData(boxfiles)
	boxes := createBoxes(boxdata, *port, hostname)

	for namespace, boxinfo := range boxes {
		for boxname, _ := range boxinfo {
			log.Println(strings.Join([]string{"Found: ", namespace, "/", boxname}, ""))
		}
	}

	bh := BoxHandler{}
	bh.Boxes = boxes

	m := mux.NewRouter()
	m.HandleFunc("/{user}/{boxname}", bh.GetBox).Methods("GET")
	m.HandleFunc("/{user}/{boxname}", bh.CheckBox).Methods("HEAD")
	//Handling downloads that look like Vagrant Cloud
	//https://vagrantcloud.com/benphegan/boot2docker/version/2/provider/vmware_desktop.box
	m.HandleFunc("/{user}/{boxname}/{version}/provider/{boxfile}", bh.DownloadBox).Methods("GET")
	m.NotFoundHandler = http.HandlerFunc(bh.NotFound)
	http.Handle("/", m)

	log.Println("Listening on port: ", *port)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)
}

type BoxHandler struct {
	Boxes map[string]map[string]Box
}

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

func getBoxList(directory string) []string {
	directoryglob := path.Join(directory, "*.box")
	files, _ := filepath.Glob(directoryglob)
	return files
}

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

func getProvider(location string) (BoxMetadata, error) {

	log.Println("Checking: ", location)
	//boxes are .tar.gz so we have to get a gzip stream and pass to tar.
	f, _ := os.Open(location)
	r, _ := gzip.NewReader(f)
	tr := tar.NewReader(r)
	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		if hdr.Name == "metadata.json" {
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
