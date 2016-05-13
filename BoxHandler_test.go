package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCanConstructBoxFromFilename(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	filenames := []string{"/tmp/benphegan-VAGRANTSLASH-development__1.0__virtualbox.box"}
	boxes := bh.getBoxData(filenames)
	assert.Equal(1, len(boxes), "We should get one box")
	assert.Equal("development", boxes[0].Boxname)
	assert.Equal("benphegan", boxes[0].Username)
	assert.Equal("virtualbox", boxes[0].Provider)
	assert.Equal("1.0", boxes[0].Version)
}

func TestCanConstructMultipleBoxexFromFilenames(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}

	filenames := []string{"/tmp/benphegan-VAGRANTSLASH-development__1.0__virtualbox.box", "/tmp/benphegan-VAGRANTSLASH-development__2.0__virtualbox.box"}
	boxes := bh.getBoxData(filenames)
	assert.Equal(2, len(boxes), "We should get two boxes")
}

func TestCanCreateTwoBoxesFromSimpleBoxes(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "1.0"},
		SimpleBox{Boxname: "uat", Username: "benphegan", Provider: "virtualbox", Version: "1.0"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal(2, len(bh.Boxes["benphegan"]))
}

func TestCreatesCorrectBoxFromSimpleBox(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal(1, len(bh.Boxes["benphegan"]))
	assert.Equal("2.0", bh.GetBox("benphegan", "dev").CurrentVersion.Version)
	assert.Equal(1, len(bh.GetBox("benphegan", "dev").CurrentVersion.Providers))
}

func TestSetsCorrectBoxAsCurrent(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "1.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "4.1"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal("4.1", bh.GetBox("benphegan", "dev").CurrentVersion.Version)
}

func TestCorrectProvidersCreated(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "vmware", Version: "1.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "4.1"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal(3, len(bh.Boxes["benphegan"]["dev"].Versions))
	//assert.Equal("vmware",bh.Boxes["benphegan"]["dev"].Versions[1].Providers[0].Name)
}

func TestCanGetBoxFileLocationForCurrent(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__2.0__virtualbox.box"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "1.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__1.0__virtualbox.box"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "4.1", Location: "/tmp/benphegan-VAGRANTSLASH-dev__4.1__virtualbox.box"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal("/tmp/benphegan-VAGRANTSLASH-dev__4.1__virtualbox.box", bh.GetBoxFileLocation("benphegan", "dev", "virtualbox", "4.1"))
}

func TestCanGetBoxFileLocationForSpecificProvider(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__2.0__virtualbox.box"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "vmware", Version: "1.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__1.0__vmware.box"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "4.1", Location: "/tmp/benphegan-VAGRANTSLASH-dev__4.1__virtualbox.box"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal("/tmp/benphegan-VAGRANTSLASH-dev__1.0__vmware.box", bh.GetBoxFileLocation("benphegan", "dev", "vmware", "1.0"))
}

func TestCanGetTwoProvidersForOneVersion(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__2.0__virtualbox.box"},
						SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "vmware", Version: "2.0", Location: "/tmp/benphegan-VAGRANTSLASH-dev__2.0__vmware.box"}}
	host := "localhost"
	bh.createBoxes(boxes, 80, &host)
	assert.Equal(2, len(bh.Boxes["benphegan"]["dev"].Versions[0].Providers))
	assert.Equal("/tmp/benphegan-VAGRANTSLASH-dev__2.0__vmware.box", bh.GetBoxFileLocation("benphegan", "dev", "vmware", "2.0"))
	assert.Equal("/tmp/benphegan-VAGRANTSLASH-dev__2.0__virtualbox.box", bh.GetBoxFileLocation("benphegan", "dev", "virtualbox", "2.0"))
}

