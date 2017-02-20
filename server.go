package main

import (
	"github.com/golang/protobuf/proto"
	"net/http"
	"fmt"
	"os"
	"log"
	"io/ioutil"
	"encoding/json"
	"math"
	"math/rand"
	"strconv"
	//"sync"
	"time"
	//"bytes"
	"flag"
)

//type base uint8

// planettype
const (
	TERRAN int32 = iota
	DESERT
	VOLCANIC
	ICE
	BARREN
	FERROUS
	GREENHOUSE
	OCEANIC
	ASTEROID
	DWARF
)

// spaceshiptype
const (
	SCOUT int32 = iota
	COLONY
	CARRIER
	BATTLESHIP
	TITAN
)

// solarsystemtype
const (
	BLUEGIANT int32 = iota
	YELLOWGIANT
	REDGIANT
	WHITEDWARF
	NEUTRONSTAR
	BLACKHOLE
)

// spaceshipstatus
const (
	OPERATIONAL int32 = iota
	LIGHTLYDAMAGED
	HEAVILYDAMAGED
	SEVERELYDAMAGED
	HANGINGBYATHREAD
)

// galaxytype
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

func absInt64(val int64) int64 {
	if val < 0 {
		val = -val
	}
	return val
}

func commaFormatUint(n uint64) string {
	in := strconv.FormatUint(n, 10)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}
	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func mulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

const mostNegative = -(mostPositive + 1)
const mostPositive = 1<<63 - 1

func signedMulOverflows(a, b int64) bool {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return false
	}
	if a == mostNegative || b == mostNegative {
		return true
	}
	c := a * b
	return c/b != a
}


func growPopulation(currentPopulation float64, foodSupply int32) float64 {
	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	referencePopulation := float64(10000)   // population if foodSupply == 0
	maxPercentChange := .005               // maximum % change allowed in single turn (.0005 is standard)

	N := currentPopulation
	println("" + commaFormatUint(uint64(N)))

	randomness := referencePopulation * (1 + rnum.NormFloat64() * .1)
	K := randomness * math.Pow(foodSupplyDbl, 3.4)
	//println("K                 = " + commaFormatUint(uint64(K)))

	logisticPercentChange := (K - N) / K

	switch {
	case logisticPercentChange < -1:
		//println("   negative prescision loss", logisticPercentChange)
		logisticPercentChange = -1
	case logisticPercentChange > 1:
		//println("   positive prescision loss", logisticPercentChange)
		logisticPercentChange = 1
	case logisticPercentChange == 0:
		//println("   zero value              ", logisticPercentChange)
		logisticPercentChange = .0000001
	}

	currentPopulation = currentPopulation + (N * maxPercentChange) * logisticPercentChange
	return currentPopulation
}

func main() {

	// define OS arguments for parsing
	inputFilePtr := flag.String("input", "data.blob", "use specified file as data source (optional)")
	outputFilePtr := flag.String("output", "", "specify json file for export (optional)")
	flag.Parse()
	inputFilename := *inputFilePtr
	outputFilename := *outputFilePtr

	// run various sanity checks and/or print error messages here
	if outputFilename == "data.blob" {
		log.Fatalln("Improper use of the 'output' parameter. 'Output' is used to specify where JSON should go. This paramter is optional.")
	}

	// initializing an empty galaxy
	theGalaxy := Galaxy{}

	if inputFilename == "data.blob" {
		in, err := ioutil.ReadFile("data.blob")
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("data.blob not found\n")
				os.Exit(1)
			} else {
				log.Fatalln("Error reading file:", err)
			}
		}
		if err := proto.Unmarshal(in, &theGalaxy); err != nil {
			log.Fatalln("Failed to parse galaxy file:", err)
		}
	} else {
		in, err := ioutil.ReadFile(inputFilename)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s - file not found!")
				os.Exit(1)
			} else {
				log.Fatalln("Error reading file:", err)
			}
		}
		if err := json.Unmarshal(in, &theGalaxy); err != nil {
			log.Fatalln("Failed to parse galaxy file:", err)
		}
	}


	i := 0
	for {
		i++
		for _, ss := range theGalaxy.Solarsystemslice {
			for _, pl := range ss.Planetslice {
				pl.Population = growPopulation(pl.Population, pl.Food)
			}
		}

		//fmt.Println(theGalaxy.Name)
		//for _, ss := range theGalaxy.Solarsystemslice {
		//    fmt.Println("  ", ss.Name)
		//    for _, pl := range ss.Planetslice {
		//        fmt.Println("    ", pl.Name)
		//        fmt.Println("      ", pl.Population)
		//    }
		//}
		if i == 100 {
			break
		}
	}



	// Create default values for Galaxy and attempt to Parse file
	//theGalaxy = Galaxy{GlobalX: 526, GlobalY: -283, Name: "Milkyway-832"}
	//someSystems := make([]*SolarSystem, 0)
	//someSystems = append(someSystems,
	//    &SolarSystem{InterX: 3293, InterY: 99, Name: "Sol-137",},
	//    &SolarSystem{InterX: -233, InterY: 10039, Name: "Xioami-137"})
	//theGalaxy.Solarsystemslice = someSystems



	// Write new galaxy to data.blob file
	protoOut, err := proto.Marshal(&theGalaxy)
	if err != nil {
		log.Fatalln("Failed to PROTOBUF encode Galaxy:", err)
	}
	if err := ioutil.WriteFile("data.blob", protoOut, 0644); err != nil {
		log.Fatalln("Failed to write Galaxy:", err)
	}


	// Write new galaxy to json file
	if outputFilename != "" {
		jsonOut, err := json.MarshalIndent(theGalaxy, "", "  ")
		if err != nil {
			log.Fatalln("Failed to JSON encode Galaxy:", err)
		}
		if err := ioutil.WriteFile(outputFilename, jsonOut, 0644); err != nil {
			log.Fatalln("Failed to write Galaxy:", err)
		}
	}

	//fmt.Println(theGalaxy)

	http.Handle("/", http.FileServer(http.Dir("D:\\OneDrive\\JSCLIENT")))
	http.ListenAndServe(":80", nil)

}
