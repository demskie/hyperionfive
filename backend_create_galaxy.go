package main

import (
	"encoding/json"
	"fmt"
	"hyperionfive/simpletree"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"regexp"

	"github.com/gogo/protobuf/proto"
)

// UnmarshalFile ...
func (gal *GalaxyStruct) UnmarshalFile(inputFile string) (rt *simpletree.Tree) {
	const fName = "(galaxy.UnmarshalFile)"
	// after deserialization we need to call the following function
	createTree := func() {
		for _, ss := range gal.Protobuf.Solarsystemslice {
			x := float64(ss.InterX)
			y := float64(ss.InterY)
			gal.Rtree.Insert(ss.Name, x, y)
		}
	}
	// parse filename to determine if it is a 'blob' file
	foundMatch, _ := regexp.MatchString(".*blob$", inputFile)
	if foundMatch {
		logInfo(fName, "protobuf unmarshalling", inputFile)
		in, err := ioutil.ReadFile(inputFile)
		if err != nil {
			if os.IsNotExist(err) {
				logError(fName, "data.blob not found")
			} else {
				logError(fName, err.Error())
			}
		}
		err = proto.Unmarshal(in, gal.Protobuf)
		checkError(err, fName)
		createTree()
		return
	}
	// parse filename to determine if it is a 'json' file
	foundMatch, _ = regexp.MatchString(".*json$", inputFile)
	if foundMatch {
		logInfo(fName, "json unmarshalling", inputFile)
		in, err := ioutil.ReadFile(inputFile)
		if err != nil {
			if os.IsNotExist(err) {
				logError(inputFile, "not found")
			} else {
				logError(fName, err.Error())
			}
		}
		err = json.Unmarshal(in, gal.Protobuf)
		checkError(err, fName)
		createTree()
		return
	}
	// if none of the above logic was successful raise and error
	logError(fName, "unknown inputfilename", inputFile)
	suddenExit()
	return
}

// Generate ...
func (gal *GalaxyStruct) Generate() {
	const fName = "(createGalaxy)"

	gal.Protobuf.Solarsystemslice = make([]*SolarSystem, totalNumSystems)
	for i := range gal.Protobuf.Solarsystemslice {
		gal.Protobuf.Solarsystemslice[i] = &SolarSystem{}
	}
	logInfo(fName, fmt.Sprintf("generating %s systems", commaFormatNum(len(gal.Protobuf.Solarsystemslice))))
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(gal.Protobuf.Solarsystemslice))
		if stopIndex <= startIndex {
			return
		}
		var ss *SolarSystem
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.Protobuf.Solarsystemslice[i]
			generateSystem(ss, rnum)
			for {
				generateEllipticalCoordinates(ss, rnum)
				gal.Rtree.Insert(ss.Name, float64(ss.InterX), float64(ss.InterY))
				objects, err := gal.Rtree.NearestNeighbors(ss.Name, 1)
				if checkError(err, fName) {
					suddenExit()
				}
				hypotenuse := math.Sqrt((objects[0].X * objects[0].X) + (objects[0].Y * objects[0].Y))
				if hypotenuse <= minimumDistanceBetweenSystems {
					gal.Rtree.Delete(ss.Name)
					continue
				}
				break
			}

		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSystem(ss *SolarSystem, rnum *rand.Rand) {
	fName := "(generateSystem)"

	// define weight for systemTypes
	weightedSystemTypes := []weightedObj{
		weightedObj{Weight: 50, Item: YELLOWGIANT},
		weightedObj{Weight: 50, Item: REDGIANT},
		weightedObj{Weight: 50, Item: BLUEGIANT},
		weightedObj{Weight: 10, Item: NEUTRONSTAR},
		weightedObj{Weight: 10, Item: WHITEDWARF},
		weightedObj{Weight: 5, Item: BLACKHOLE},
	}
	// generate systemType
	result, err := weightedChoice(weightedSystemTypes, rnum)
	checkError(err, fName)
	ss.Type = result.Item.(int32)

	// generate name
	canidateName := generateName(SOLARSYSTEMNAME, ss.Type)
	ss.Name = canidateName.GetFullName()

	// generate number of objects
	var numObj int64
	if ss.Type == YELLOWGIANT {
		numObj = int64(math.Abs(rnum.NormFloat64()*7) + 20)
	} else if ss.Type == REDGIANT {
		numObj = int64(math.Abs(rnum.NormFloat64()*3) + 25)
	} else if ss.Type == BLUEGIANT {
		numObj = int64(math.Abs(rnum.NormFloat64()*7) + 20)
	} else if ss.Type == NEUTRONSTAR {
		numObj = int64(math.Abs(rnum.NormFloat64()*7) + 10)
	} else if ss.Type == WHITEDWARF {
		numObj = int64(0)
	} else if ss.Type == BLACKHOLE {
		numObj = int64(rnum.NormFloat64()*10 + 30)
		if numObj < int64(11) {
			numObj = int64(11)
		} else if numObj > int64(72) {
			numObj = int64(52)
		}
	} else {
		logError(fName, "invalid system.Type:", fmt.Sprint(ss.Type))
		suddenExit()
	}
	if numObj > int64(72) {
		numObj = int64(53)
	}

	// define weight for this object based on SolarSystemType
	var weightedObjTypes []weightedObj
	if ss.Type == YELLOWGIANT {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == REDGIANT {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == BLUEGIANT {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == NEUTRONSTAR {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == WHITEDWARF {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == BLACKHOLE {
		weightedObjTypes = []weightedObj{
			weightedObj{Weight: 50, Item: TERRAN},
			weightedObj{Weight: 50, Item: DESERT},
			weightedObj{Weight: 50, Item: VOLCANIC},
			weightedObj{Weight: 10, Item: ICE},
			weightedObj{Weight: 10, Item: BARREN},
			weightedObj{Weight: 5, Item: FERROUS},
			weightedObj{Weight: 5, Item: GREENHOUSE},
			weightedObj{Weight: 5, Item: OCEANIC},
			weightedObj{Weight: 5, Item: LARGEASTEROID},
			weightedObj{Weight: 5, Item: DWARF},
			weightedObj{Weight: 5, Item: ASTEROIDBELT},
			weightedObj{Weight: 5, Item: PLASMASTORM},
		}
	}

	// start building objects
	ss.Solarobjectslice = []*SolarObject{}
	for i := 0; i < int(numObj); i++ {
		newObj := &SolarObject{}
		// generate type
		result, err := weightedChoice(weightedObjTypes, rnum)
		checkError(err, fName)
		newObj.Type = result.Item.(int32)
		// generate name
		canidateName := generateName(SOLARSYSTEMNAME, newObj.Type)
		newObj.Name = canidateName.GetFullName()
		// insert parentSystemName
		newObj.ParentSystemName = ss.Name
		// append result
		ss.Solarobjectslice = append(ss.Solarobjectslice, newObj)
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSpiralCoordinates(ss *SolarSystem, numArms int, rnum *rand.Rand) {
	roundFloat := func(val float64) float64 {
		if val < 0 {
			return float64(int64(val - 0.5))
		}
		return float64(int64(val + 0.5))
	}
	fDeg2Rad := math.Pi / 180
	fArmAngle := math.Mod(360/float64(numArms), 360)
	fAngularSpread := 90 / float64(numArms)
	fNumArms := float64(numArms)
	fK := float64(1)
	fR := float64(rnum.Int63n(galaxyRadius))
	fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(getPosOrNegOne(rnum))
	fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterX = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
	fR = float64(rnum.Int63n(galaxyRadius))
	fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(getPosOrNegOne(rnum))
	fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterY = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
}

func generateEllipticalCoordinates(ss *SolarSystem, rnum *rand.Rand) {
	var A2, B2, C float64
	for {
		ss.InterX = rnum.Int63() * int64(getPosOrNegOne(rnum))
		ss.InterY = rnum.Int63() * int64(getPosOrNegOne(rnum))
		A2 = float64(ss.InterX * ss.InterX)
		B2 = float64(ss.InterY * ss.InterY)
		C = math.Sqrt(A2 + B2)
		if C >= galaxyRadius {
			continue
		}
		break
	}
}
