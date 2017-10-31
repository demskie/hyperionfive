package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var loadedVerbs []string
var loadedNouns []string

// NamedObject ...
type NamedObject struct {
	Verb string
	Noun string
	Hex  string
}

func replaceStringAtIndex(str string, replacement rune, index int) string {
	out := []rune(str)
	out[index] = replacement
	return string(out)
}

// LoadAllNames ...
func (obj *NamedObject) LoadAllNames() (verbs []string, nouns []string) {
	if len(loadedVerbs) != 0 {
		verbs = loadedVerbs
	} else {
		rawBytes, err := ioutil.ReadFile("names_nouns.txt")
		if checkError(err, "Unable to load 'names_nouns.txt'") {
			os.Exit(1)
		}
		rawString := string(rawBytes[:])
		verbs = strings.Split(rawString, "\n")
		for i, s := range verbs {
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			if s != "" {
				verbs[i] = strings.Title(s)
			}
		}
	}
	if len(loadedNouns) != 0 {
		nouns = loadedNouns
	} else {
		rawBytes, err := ioutil.ReadFile("names_verbs.txt")
		if checkError(err, "Unable to load 'names_verbs.txt'") {
			os.Exit(1)
		}
		rawString := string(rawBytes[:])
		verbs = strings.Split(rawString, "\n")
		for i, s := range nouns {
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			if s != "" {
				verbs[i] = strings.Title(s)
			}
		}
	}
	return verbs, nouns
}

// GetFullName ...
func (obj *NamedObject) GetFullName() string {
	return obj.Verb + obj.Noun + obj.Hex
}

// identifying objectType for naming generator
const (
	GALAXYNAME = iota
	SOLARSYSTEMNAME
	PLANETARYOBJECTNAME
)

func generateName(nameType int32, objType int32) NamedObject {
	newName := NamedObject{}
	allVerbs, allNames := newName.LoadAllNames()
	rnum := getCryptoRnum()
	idx := rnum.Intn(len(allVerbs))
	newName.Verb = allVerbs[idx]
	idx = rnum.Intn(len(allNames))
	newName.Noun = allNames[idx]
	newName.Hex = strconv.FormatUint(rnum.Uint64(), 16)
	return newName
}
