package main

import (
	"github.com/BenPhegan/vagrantshadow/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"testing"
)

func TestCanConstructBoxFromFilename(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	filenames := []string{"/tmp/benphegan-VAGRANTSLASH-development__virtualbox__1.0.box"}
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

	filenames := []string{"/tmp/benphegan-VAGRANTSLASH-development__virtualbox__1.0.box", "/tmp/benphegan-VAGRANTSLASH-development__virtualbox__2.0.box"}
	boxes := bh.getBoxData(filenames)
	assert.Equal(2, len(boxes), "We should get two boxes")
}

func TestCanCreateTwoBoxesFromSimpleBoxes(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "1.0"},
		SimpleBox{Boxname: "uat", Username: "benphegan", Provider: "virtualbox", Version: "1.0"}}
	host := "localhost"
	realBoxes := bh.createBoxes(boxes, 80, &host)
	assert.Equal(2, len(realBoxes["benphegan"]))
}

func TestCreatesCorrectBoxFromSimpleBox(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0"}}
	host := "localhost"
	realBoxes := bh.createBoxes(boxes, 80, &host)
	assert.Equal(1, len(realBoxes["benphegan"]))
	assert.Equal("2.0", realBoxes["benphegan"]["dev"].CurrentVersion.Version)
	assert.Equal(1, len(realBoxes["benphegan"]["dev"].CurrentVersion.Providers))
}

func TestSetsCorrectBoxAsCurrent(t *testing.T) {
	assert := assert.New(t)
	bh := BoxHandler{}
	boxes := []SimpleBox{SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "2.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "1.0"},
		SimpleBox{Boxname: "dev", Username: "benphegan", Provider: "virtualbox", Version: "4.1"}}
	host := "localhost"
	realBoxes := bh.createBoxes(boxes, 80, &host)
	assert.Equal("4.1", realBoxes["benphegan"]["dev"].CurrentVersion.Version)

}
