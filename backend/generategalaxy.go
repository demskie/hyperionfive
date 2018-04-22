package backend

import (
	"encoding/json"
	"fmt"
	"hyperionfive/intutil"
	"hyperionfive/names"
	"hyperionfive/randutil"
	"hyperionfive/simpletree"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"regexp"
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
)

// UnmarshalFile ...
func (gal *GalaxyStruct) UnmarshalFile() (rt *simpletree.Tree) {
	const fName = "(galaxy.UnmarshalFile)"
	// after deserialization we need to call the following function
	createTree := func() {
		for _, ss := range gal.Protobuf.Solarsystemslice {
			x := float64(ss.InterX)
			y := float64(ss.InterY)
			gal.Rtree.Insert(ss.NameInt, x, y)
		}
	}
	// parse filename to determine if it is a 'blob' file
	foundMatch, _ := regexp.MatchString(".*blob$", args.InputFile)
	if foundMatch {
		log.Info(fName, "protobuf unmarshalling", args.InputFile)
		in, err := ioutil.ReadFile(args.InputFile)
		if err != nil {
			if os.IsNotExist(err) {
				log.Error(fName, "data.blob not found")
			} else {
				log.Error(fName, err.Error())
			}
		}
		err = proto.Unmarshal(in, gal.Protobuf)
		log.CheckError(err, fName)
		createTree()
		return
	}
	// parse filename to determine if it is a 'json' file
	foundMatch, _ = regexp.MatchString(".*json$", args.InputFile)
	if foundMatch {
		log.Info(fName, "json unmarshalling", args.InputFile)
		in, err := ioutil.ReadFile(args.InputFile)
		if err != nil {
			if os.IsNotExist(err) {
				log.Error(args.InputFile, "not found")
			} else {
				log.Error(fName, err.Error())
			}
		}
		err = json.Unmarshal(in, gal.Protobuf)
		log.CheckError(err, fName)
		createTree()
		return
	}
	// if none of the above logic was successful raise and error
	log.Error(fName, "unknown inputfilename", args.InputFile)
	log.SuddenExit()
	return
}

// Generate ...
func (gal *GalaxyStruct) Generate() {
	const fName = "(createGalaxy)"
	gal.Protobuf.Solarsystemslice = make([]*SolarSystem, totalNumSystems)
	for i := range gal.Protobuf.Solarsystemslice {
		gal.Protobuf.Solarsystemslice[i] = &SolarSystem{}
	}
	badSystems := int32(0)
	log.Info(fName, fmt.Sprintf("generating %v systems",
		intutil.CommaFormatNum(len(gal.Protobuf.Solarsystemslice))))
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(gal.Protobuf.Solarsystemslice))
		if stopIndex <= startIndex {
			return
		}
		var ss *SolarSystem
		var x1, y1, x2, y2 float64
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.Protobuf.Solarsystemslice[i]
			generateSystem(ss, rnum, gal.SystemAndPlanetNames)
			for {
				generateEllipticalCoordinates(ss, rnum)
				x1, y1 = float64(ss.InterX), float64(ss.InterY)
				gal.Rtree.Insert(ss.NameInt, x1, y1)
				objects, err := gal.Rtree.NearestNeighbors(ss.NameInt, 1)
				if log.CheckError(err, fName) {
					log.SuddenExit()
				}
				if len(objects) == 0 {
					continue
				}
				x2, y2 = objects[0].X, objects[0].Y
				hypotenuse := math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
				if hypotenuse <= minimumDistanceBetweenSystems {
					gal.Rtree.Delete(ss.NameInt)
					atomic.AddInt32(&badSystems, 1)
					continue
				}
				break
			}
		}
	})
	log.Info(fName, fmt.Sprintf("regenerated %v systems", intutil.CommaFormatNum(badSystems)))
	log.Info(fName, "generating system links")
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(gal.Protobuf.Solarsystemslice))
		if stopIndex <= startIndex {
			return
		}
		var ss *SolarSystem
		var normalDistVal int
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.Protobuf.Solarsystemslice[i]
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			normalDistVal = 4 + int(rnum.NormFloat64()*1.2)
			if normalDistVal < 1 {
				normalDistVal = 1
			} else if normalDistVal > 9 {
				normalDistVal = 9
			}
			ss.ConnectedSolarSystems = make([]uint64, normalDistVal)
			neighbors, err := gal.Rtree.NearestNeighbors(ss.NameInt, normalDistVal)
			if log.CheckError(err, fName) {
				log.SuddenExit()
			}
			for i, neighbor := range neighbors {
				ss.ConnectedSolarSystems[i] = neighbor.ID
			}
		}
	})
}

func generateSystem(ss *SolarSystem, rnum *rand.Rand, nameGen *names.NameGenerator) {
	const fName = "(generateSystem)"

	// define weight for systemTypes
	weightedSystemTypes := []randutil.WeightedObj{
		randutil.WeightedObj{Weight: 50, Item: YELLOWGIANT},
		randutil.WeightedObj{Weight: 50, Item: REDGIANT},
		randutil.WeightedObj{Weight: 50, Item: BLUEGIANT},
		randutil.WeightedObj{Weight: 10, Item: NEUTRONSTAR},
		randutil.WeightedObj{Weight: 10, Item: WHITEDWARF},
		randutil.WeightedObj{Weight: 5, Item: BLACKHOLE},
	}
	// generate systemType
	result, err := randutil.WeightedChoice(weightedSystemTypes, rnum)
	log.CheckError(err, fName)
	ss.Type = result.Item.(int32)

	// generate name
	ss.NameInt = nameGen.GenerateName().GetNameUint()

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
		log.Error(fName, fmt.Sprintf("invalid system.Type: %v", ss.Type))
		log.SuddenExit()
	}
	if numObj > int64(72) {
		numObj = int64(53)
	}

	// define weight for this object based on SolarSystemType
	var weightedObjTypes []randutil.WeightedObj
	if ss.Type == YELLOWGIANT {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == REDGIANT {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == BLUEGIANT {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == NEUTRONSTAR {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == WHITEDWARF {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	} else if ss.Type == BLACKHOLE {
		weightedObjTypes = []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Item: TERRAN},
			randutil.WeightedObj{Weight: 50, Item: DESERT},
			randutil.WeightedObj{Weight: 50, Item: VOLCANIC},
			randutil.WeightedObj{Weight: 10, Item: ICE},
			randutil.WeightedObj{Weight: 10, Item: BARREN},
			randutil.WeightedObj{Weight: 5, Item: FERROUS},
			randutil.WeightedObj{Weight: 5, Item: GREENHOUSE},
			randutil.WeightedObj{Weight: 5, Item: OCEANIC},
			randutil.WeightedObj{Weight: 5, Item: LARGEASTEROID},
			randutil.WeightedObj{Weight: 5, Item: DWARF},
			randutil.WeightedObj{Weight: 5, Item: ASTEROIDBELT},
			randutil.WeightedObj{Weight: 5, Item: PLASMASTORM},
		}
	}

	// start building objects
	ss.Solarobjectslice = []*SolarObject{}
	for i := 0; i < int(numObj); i++ {
		newObj := &SolarObject{}
		// generate type
		result, err := randutil.WeightedChoice(weightedObjTypes, rnum)
		log.CheckError(err, fName)
		newObj.Type = result.Item.(int32)
		// generate name
		newObj.NameInt = nameGen.GenerateName().GetNameUint()
		// insert parentSystemName
		newObj.ParentSystemName = ss.NameInt
		// append result
		ss.Solarobjectslice = append(ss.Solarobjectslice, newObj)
	}

}

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
	fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterX = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
	fR = float64(rnum.Int63n(galaxyRadius))
	fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterY = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
}

func generateEllipticalCoordinates(ss *SolarSystem, rnum *rand.Rand) {
	var X, Y float64
	for {
		ss.InterX = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		ss.InterY = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		X = float64(ss.InterX * ss.InterX)
		Y = float64(ss.InterY * ss.InterY)
		if math.Sqrt(X+Y) >= galaxyRadius {
			continue
		}
		break
	}
}
