package main

import (
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/mcuadros/go-version"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sort"
)

type BoxHandler struct {
	Boxes          map[string]map[string]Box
	Directories    []string
	TemplateString string
	Hostname       string
	Port           int
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

func (bh *BoxHandler) BoxRegex() string {
	return `(?P<owner>\w*)-VAGRANTSLASH-(?P<boxname>[a-zA-Z0-9]*)__(?P<version>[a-zA-Z0-9\.-]*)__(?P<provider>[a-zA-Z0-9]*).box`
}

func (bh *BoxHandler) BoxAvailable(username string, boxname string) bool {
	return (bh.Boxes[username][boxname].Username != "")
}

func (bh *BoxHandler) GetBoxFileLocation(username string, boxName string, provider string, version string) string {
	boxList := bh.Boxes[username][boxName]
	for _, box := range boxList.Versions {
		if box.Version == version {
			for _, boxprovider := range box.Providers {
				if boxprovider.Name == provider {
					return boxprovider.LocalBoxFile
				}
			}
		}
	}
	return ""
}

func (bh *BoxHandler) GetBox(user string, boxName string) Box {
	return bh.Boxes[user][boxName]
}

func (bh *BoxHandler) PopulateBoxes(directories []string, port *int, hostname *string) {
	log.Println("Populating boxes..")
	absolutedirectories := []string{}
	for _, d := range directories {
		if !path.IsAbs(d) {
			wd, _ := os.Getwd()
			absolute := path.Clean(path.Join(wd, d))
			absolutedirectories = append(absolutedirectories, absolute)
		} else {
			absolutedirectories = append(absolutedirectories, d)
		}
	}
	bh.Directories = absolutedirectories
	boxfiles := getBoxList(absolutedirectories)
	boxdata := bh.getBoxData(boxfiles)
	bh.createBoxes(boxdata, *port, hostname)

	for _, boxinfo := range bh.Boxes {
		for boxname, box := range boxinfo {
			for _, version := range box.Versions {
				for _, provider := range version.Providers {
					log.Println("Found " + box.Username + "/" + boxname + "/" + version.Version + "/" + provider.Name)
				}
			}
		}
	}
}

// getBoxList returns a list of .box files in the directories provided.
// Returns full path
func getBoxList(directories []string) []string {
	boxes := []string{}
	for _, d := range directories {
		log.Println("Checking for files in: " + d)
		directoryglob := path.Join(d, "*.box")
		files, _ := filepath.Glob(directoryglob)
		boxes = append(boxes, files...)
	}
	return boxes
}

//getBoxData returns an array of SimpleBox objects based on Vagrant box files
func (bh *BoxHandler) getBoxData(boxfiles []string) []SimpleBox {
	results := []SimpleBox{}
	var myExp = regexp.MustCompile(bh.BoxRegex())
	for _, b := range boxfiles {
		matches := myExp.FindStringSubmatch(filepath.Base(b))
		if matches == nil || len(matches) != 5 {
			log.Println("Could not match metadata from filename: " + filepath.Base(b))
			return results
		}
		newbox := SimpleBox{Username: matches[1], Boxname: matches[2], Location: b, Provider: matches[4], Version: matches[3]}
		results = append(results, newbox)
	}

	return results
}

//Creates the data structure used to provide box data to Vagrant
func (bh *BoxHandler) createBoxes(sb []SimpleBox, port int, hostname *string) {
	boxes := make(map[string]map[string]Box)
	for _, b := range sb {

		box := boxes[b.Username][b.Boxname]
		if box.Name == "" {
			box = Box{}
			box.Name = b.Username + "/" + b.Boxname
			box.Username = b.Username
			box.Private = false
		}

		provider := Provider{}
		provider.Name = b.Provider
		provider.Hosted = "true"
		provider.DownloadUrl = "http://" + *hostname + ":" + strconv.Itoa(port) + "/" + b.Username + "/" + b.Boxname + "/" + b.Version + "/" + b.Provider + "/" + b.Provider + ".box"
		provider.Url = provider.DownloadUrl
		provider.LocalBoxFile = b.Location

		if len(box.Versions) > 0 {
			providerAppended := false
			for i, v := range box.Versions {
				if v.Version == b.Version {
					box.Versions[i].Providers = append(box.Versions[i].Providers, provider)
					providerAppended = true
				}
			}
			
			if providerAppended == false {
				newversion := Version{}
				newversion.Status = "active"
				newversion.Version = b.Version
				newversion.Providers = []Provider{provider}
				box.Versions = append(box.Versions, newversion)
			}
		} else {
			newversion := Version{}
			newversion.Status = "active"
			newversion.Version = b.Version
			newversion.Providers = []Provider{provider}
			box.Versions = []Version{newversion}
		}

		if boxes[b.Username] == nil {
			boxes[b.Username] = make(map[string]Box)
		}

		sort.Slice(box.Versions[:], func(i, j int) bool {
			return version.Compare(box.Versions[i].Version, box.Versions[j].Version, ">")
		})
		box.CurrentVersion = &box.Versions[0]

		boxes[b.Username][b.Boxname] = box
	}

	bh.Boxes = boxes
}
