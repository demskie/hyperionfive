package names

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/logging"
)

// NameStruct ...
type NameStruct struct {
	verb uint16
	noun uint16
	hex  uint32
}

// GetNameUint ...
func (n *NameStruct) GetNameUint() uint64 {
	byteSlice := make([]byte, 8)
	binary.LittleEndian.PutUint16(byteSlice[0:2], n.verb)
	binary.LittleEndian.PutUint16(byteSlice[2:4], n.noun)
	binary.LittleEndian.PutUint32(byteSlice[4:8], n.hex)
	return binary.LittleEndian.Uint64(byteSlice)
}

// NameGenerator ...
type NameGenerator struct {
	mtx         *sync.Mutex
	loadedVerbs []string
	loadedNouns []string
	generated   map[uint64]NameStruct
}

// NewNameGenerator ...
func NewNameGenerator(args *arguments.Args, log *logging.Logger) *NameGenerator {
	const fName = "(names.Load)"
	exPath, err := os.Executable()
	if log.CheckError(err, fName) {
		log.SuddenExit()
	}
	dir := filepath.Dir(exPath)
	verbs, nouns := []string{}, []string{}
	for _, filename := range []string{"names_verbs.txt", "names_nouns.txt"} {
		rawBytes, err := ioutil.ReadFile(filepath.Join(dir, "websiteroot", filename))
		if log.CheckError(err, fName) {
			log.SuddenExit()
		}
		rawString := string(rawBytes[:])
		lines := strings.Split(rawString, "\n")
		for i, s := range lines {
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			lastLineIdx := len(lines) - 1
			if s == "" && i != lastLineIdx {
				log.Error(fName, filename, "has an empty string on line", fmt.Sprint(i))
			} else if filename == "names_verbs.txt" {
				verbs = append(verbs, strings.Title(s))
			} else {
				nouns = append(nouns, strings.Title(s))
			}
		}
	}
	nameGen := &NameGenerator{
		loadedVerbs: verbs,
		loadedNouns: nouns,
		generated:   map[uint64]NameStruct{},
	}
	return nameGen
}

// GenerateName ...
func (n *NameGenerator) GenerateName(rnum *rand.Rand) (newName *NameStruct) {
	return &NameStruct{
		verb: uint16(rnum.Intn(len(n.loadedVerbs))),
		noun: uint16(rnum.Intn(len(n.loadedNouns))),
		hex:  uint32(rnum.Int31()),
	}
}

// AddToIndex ...
func (n *NameGenerator) AddToIndex(name *NameStruct) {
	n.mtx.Lock()
	n.generated[name.GetNameUint()] = *name
	n.mtx.Unlock()
}

// GetNameStruct ...
func (n *NameGenerator) GetNameStruct(num uint64) (*NameStruct, bool) {
	val, exists := n.generated[num]
	return &val, exists
}

// GetNameString ...
func (n *NameGenerator) GetNameString(obj NameStruct) string {
	result := n.loadedVerbs[obj.verb] + n.loadedNouns[obj.noun] + strconv.FormatInt(int64(obj.hex), 16)
	return result
}
