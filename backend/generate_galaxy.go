package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"regexp"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/randutil"
	"github.com/demskie/spatialindex"

	"github.com/gogo/protobuf/proto"
)

// UnmarshalFile ...
func (gal *GalaxyStruct) UnmarshalFile() (rt *spatialindex.Tree) {
	const fName = "(galaxy.UnmarshalFile)"
	// after deserialization we need to call the following function
	createTree := func() {
		for ssName, ss := range gal.Protobuf.System {
			x := float64(ss.InterX)
			y := float64(ss.InterY)
			gal.Rtree.Add(ssName, x, y)
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

// TODO // unorganized - need to refactor

// Generate ...
func (gal *GalaxyStruct) Generate() {
	const fName = "(createGalaxy)"
	log.Info(fName, fmt.Sprintf("generating %v systems",
		intutil.CommaFormatNum(totalNumSystems)))
	rnum := randutil.CreateUniqueMathRnum()
	for len(gal.Protobuf.System) < totalNumSystems {
		ss := &gamestate.System{}

		// define weight for systemTypes
		weightedSystemTypes := []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Object: gamestate.YELLOWGIANT},
			randutil.WeightedObj{Weight: 50, Object: gamestate.REDGIANT},
			randutil.WeightedObj{Weight: 50, Object: gamestate.BLUEGIANT},
			randutil.WeightedObj{Weight: 10, Object: gamestate.NEUTRONSTAR},
			randutil.WeightedObj{Weight: 10, Object: gamestate.WHITEDWARF},
			randutil.WeightedObj{Weight: 5, Object: gamestate.BLACKHOLE},
		}

		// generate systemType
		result, err := randutil.WeightedChoice(weightedSystemTypes, rnum)
		log.CheckError(err, fName)
		ss.SystemType = result.Object.(int32)

		// generate name
		rawName := gal.SystemAndPlanetNames.GenerateName(rnum)
		ssName := rawName.GetNameUint()
		gal.SystemAndPlanetNames.AddToIndex(rawName)

		// generate number of planets
		var numPlanets int
		if ss.SystemType == gamestate.YELLOWGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if ss.SystemType == gamestate.REDGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*3) + 25)
		} else if ss.SystemType == gamestate.BLUEGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if ss.SystemType == gamestate.NEUTRONSTAR {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 10)
		} else if ss.SystemType == gamestate.WHITEDWARF {
			numPlanets = 0
		} else if ss.SystemType == gamestate.BLACKHOLE {
			numPlanets = int(rnum.NormFloat64()*10 + 30)
			if numPlanets < 11 {
				numPlanets = 11
			} else if numPlanets > 72 {
				numPlanets = 52
			}
		} else {
			log.Error(fName, fmt.Sprintf("invalid system.Type: %v", ss.SystemType))
			log.SuddenExit()
		}
		if numPlanets > 72 {
			numPlanets = 53
		}

		// define weight for this object based on SolarSystemType
		var weightedObjTypes []randutil.WeightedObj
		if ss.SystemType == gamestate.YELLOWGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		} else if ss.SystemType == gamestate.REDGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		} else if ss.SystemType == gamestate.BLUEGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		} else if ss.SystemType == gamestate.NEUTRONSTAR {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		} else if ss.SystemType == gamestate.WHITEDWARF {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		} else if ss.SystemType == gamestate.BLACKHOLE {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: gamestate.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: gamestate.DESERT},
				randutil.WeightedObj{Weight: 50, Object: gamestate.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: gamestate.ICE},
				randutil.WeightedObj{Weight: 10, Object: gamestate.BARREN},
				randutil.WeightedObj{Weight: 5, Object: gamestate.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: gamestate.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: gamestate.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: gamestate.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: gamestate.DWARF},
			}
		}

		// generate coordinates
		for {
			generateEllipticalCoordinates(ss, rnum)
			x1, y1 := float64(ss.InterX), float64(ss.InterY)
			gal.Rtree.Add(ssName, x1, y1)
			objects, err := gal.Rtree.NearestNeighbors(ssName, 1)
			if err != spatialindex.ErrNotEnoughNeighbors && log.CheckError(err, fName) {
				log.SuddenExit()
			}
			if len(gal.Protobuf.System) > 0 {
				x2, y2 := objects[0].X, objects[0].Y
				if math.Hypot(x1-x2, y1-y2) <= gamestate.MinimumDistanceBetweenSystems {
					gal.Rtree.Delete(ssName)
					continue
				}
			}
			ss.InterX = int64(x1)
			ss.InterY = int64(y1)
			break
		}

		// start building planets
		for len(ss.ChildPlanet) < numPlanets {
			newPlanet := &gamestate.Planet{}
			newPlanet.ParentSystemUint = ssName
			// generate type
			result, err := randutil.WeightedChoice(weightedObjTypes, rnum)
			log.CheckError(err, fName)
			newPlanet.PlanetType = result.Object.(int32)
			// generate name
			rawPlanetName := gal.SystemAndPlanetNames.GenerateName(rnum)
			ss.ChildPlanet[rawPlanetName.GetNameUint()] = gamestate.GlobalNullMessage
			gal.SystemAndPlanetNames.AddToIndex(rawPlanetName)
			// add planet
			gal.Protobuf.Planet[rawPlanetName.GetNameUint()] = newPlanet
		}
	}
	log.Info(fName, "generating system links")
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartStopForCore(
			coreNum, args.NumberOfCores, len(gal.SystemSlice))
		if stopIndex <= startIndex {
			return
		}
		var ss *gamestate.System
		var normalDistVal int
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.SystemSlice[i]
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			normalDistVal = 4 + int(rnum.NormFloat64()*1.2)
			if normalDistVal < 1 {
				normalDistVal = 1
			} else if normalDistVal > 9 {
				normalDistVal = 9
			}
			neighbors, err := gal.Rtree.NearestNeighbors(gal.SystemNames[i], normalDistVal)
			if err != spatialindex.ErrNotEnoughNeighbors &&
				log.CheckError(err, fName) {
				log.SuddenExit()
			}
			for _, neighbor := range neighbors {
				ss.NearbySystems[neighbor.ID] = gamestate.GlobalNullMessage
			}
		}
	})
}

func generateSpiralCoordinates(ss *gamestate.System, numArms int, rnum *rand.Rand) {
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
	fR := float64(rnum.Int63n(gamestate.GalaxyRadius))
	fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterX = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
	fR = float64(rnum.Int63n(gamestate.GalaxyRadius))
	fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterY = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
}

func generateEllipticalCoordinates(ss *gamestate.System, rnum *rand.Rand) {
	var X, Y float64
	for {
		ss.InterX = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		ss.InterY = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		X = float64(ss.InterX * ss.InterX)
		Y = float64(ss.InterY * ss.InterY)
		if math.Sqrt(X+Y) >= gamestate.GalaxyRadius {
			continue
		}
		break
	}
}
