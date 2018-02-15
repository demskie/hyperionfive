package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

var loadedGalaxyNames map[string]NamedObject
var loadedSystemNames map[string]NamedObject

func replaceStringAtIndex(str string, replacement rune, index int) string {
	out := []rune(str)
	out[index] = replacement
	return string(out)
}

func init() {
	const fName = "(loadAllNames)"
	exPath, err := os.Executable()
	if checkError(err, fName) {
		suddenExit()
	}
	dir := filepath.Dir(exPath)
	for _, filename := range []string{"names_verbs.txt", "names_nouns.txt"} {
		rawBytes, err := ioutil.ReadFile(filepath.Join(dir, filename))
		if checkError(err, fName) {
			suddenExit()
		}
		rawString := string(rawBytes[:])
		lines := strings.Split(rawString, "\n")
		for i, s := range lines {
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			lastLineIdx := len(lines) - 1
			if s == "" && i != lastLineIdx {
				logError(fName, filename, "has an empty string on line", fmt.Sprint(i))
			} else if filename == "names_verbs.txt" {
				loadedVerbs = append(loadedVerbs, strings.Title(s))
			} else {
				loadedNouns = append(loadedNouns, strings.Title(s))
			}
		}
	}
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
	rnum := getCryptoRnum()
	idx := rnum.Intn(len(loadedVerbs))
	newName.Verb = loadedVerbs[idx]
	idx = rnum.Intn(len(loadedNouns))
	newName.Noun = loadedNouns[idx]
	newName.Hex = strconv.FormatUint(rnum.Uint64(), 16)[0:6]
	return newName
}
