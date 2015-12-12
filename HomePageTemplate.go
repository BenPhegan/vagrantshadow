package main

import (
	"io/ioutil"
	"log"
	"os"
)

type HomePageTemplate struct {
	TemplateString string
	BoxHandler     *BoxHandler
}

func (ht *HomePageTemplate) GetDefaultTemplateString() string {
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

func (ht *HomePageTemplate) OutputTemplateString(location string) {
	if _, err := os.Stat(location); os.IsNotExist(err) {
		log.Println("Writing out default home template file: " + location)
		err := ioutil.WriteFile(location, []byte(ht.GetDefaultTemplateString()), 0644)
		if err != nil {
			log.Println("Failed to write default template to: " + location + " - " + err.Error())
		}
	} else {
		log.Println("Default template exists on disk already")
	}
}

func (ht *HomePageTemplate) GetTemplateString(location string) string {
	if _, err := os.Stat(location); err == nil {
		log.Println("Found template file: " + location)
		templatetext, err := ioutil.ReadFile(location)
		if err != nil {
			log.Println("Could not load template: " + location)
			return ht.GetDefaultTemplateString()
		}
		template := string(templatetext)
		return template
	}
	return ht.GetDefaultTemplateString()
}
