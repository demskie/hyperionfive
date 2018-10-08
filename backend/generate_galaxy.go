package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"regexp"

	"github.com/demskie/hyperionfive/backendmessage"
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
	uniqueNames := map[uint64]struct{}{}
	generateNameUint := func() uint64 {
		var nameUint uint64
		notFound := true
		for notFound {
			nameUint = rnum.Uint64()
			_, notFound = uniqueNames[nameUint]
		}
		uniqueNames[nameUint] = struct{}{}
		return nameUint
	}
	for len(gal.Protobuf.System) < totalNumSystems {
		ss := &backendmessage.System{}

		// define weight for systemTypes
		weightedSystemTypes := []randutil.WeightedObj{
			randutil.WeightedObj{Weight: 50, Object: backendmessage.YELLOWGIANT},
			randutil.WeightedObj{Weight: 50, Object: backendmessage.REDGIANT},
			randutil.WeightedObj{Weight: 50, Object: backendmessage.BLUEGIANT},
			randutil.WeightedObj{Weight: 10, Object: backendmessage.NEUTRONSTAR},
			randutil.WeightedObj{Weight: 10, Object: backendmessage.WHITEDWARF},
			randutil.WeightedObj{Weight: 5, Object: backendmessage.BLACKHOLE},
		}

		// generate systemType
		result, err := randutil.WeightedChoice(weightedSystemTypes, rnum)
		log.CheckError(err, fName)
		ss.SystemType = result.Object.(int32)

		// generate nameUint
		ssName := generateNameUint()

		// generate number of planets
		var numPlanets int
		if ss.SystemType == backendmessage.YELLOWGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if ss.SystemType == backendmessage.REDGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*3) + 25)
		} else if ss.SystemType == backendmessage.BLUEGIANT {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if ss.SystemType == backendmessage.NEUTRONSTAR {
			numPlanets = int(math.Abs(rnum.NormFloat64()*7) + 10)
		} else if ss.SystemType == backendmessage.WHITEDWARF {
			numPlanets = 0
		} else if ss.SystemType == backendmessage.BLACKHOLE {
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
		if ss.SystemType == backendmessage.YELLOWGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
			}
		} else if ss.SystemType == backendmessage.REDGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
			}
		} else if ss.SystemType == backendmessage.BLUEGIANT {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
			}
		} else if ss.SystemType == backendmessage.NEUTRONSTAR {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
			}
		} else if ss.SystemType == backendmessage.WHITEDWARF {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
			}
		} else if ss.SystemType == backendmessage.BLACKHOLE {
			weightedObjTypes = []randutil.WeightedObj{
				randutil.WeightedObj{Weight: 50, Object: backendmessage.TERRAN},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.DESERT},
				randutil.WeightedObj{Weight: 50, Object: backendmessage.VOLCANIC},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.ICE},
				randutil.WeightedObj{Weight: 10, Object: backendmessage.BARREN},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.FERROUS},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.GREENHOUSE},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.OCEANIC},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.LARGEASTEROID},
				randutil.WeightedObj{Weight: 5, Object: backendmessage.DWARF},
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
				if math.Hypot(x1-x2, y1-y2) <= backendmessage.MinimumDistanceBetweenSystems {
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
			newPlanet := &backendmessage.Planet{}
			newPlanet.ParentSystemUint = ssName
			// generate type
			result, err := randutil.WeightedChoice(weightedObjTypes, rnum)
			log.CheckError(err, fName)
			newPlanet.PlanetType = result.Object.(int32)
			// generate name
			nameUint := generateNameUint()
			ss.ChildPlanet[nameUint] = backendmessage.GlobalNullMessage
			// add planet
			gal.Protobuf.Planet[nameUint] = newPlanet
		}
	}
	log.Info(fName, "generating system links")
	for systemName, system := range gal.Protobuf.System {
		// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
		normalDistVal := 4 + int(gal.SingleRnum.NormFloat64()*1.2)
		if normalDistVal < 1 {
			normalDistVal = 1
		} else if normalDistVal > 9 {
			normalDistVal = 9
		}
		neighbors, err := gal.Rtree.NearestNeighbors(systemName, normalDistVal)
		if err != spatialindex.ErrNotEnoughNeighbors &&
			log.CheckError(err, fName) {
			log.SuddenExit()
		}
		for _, neighbor := range neighbors {
			system.NearbySystems[neighbor.ID] = backendmessage.GlobalNullMessage
		}
	}
}

func generateSpiralCoordinates(ss *backendmessage.System, numArms int, rnum *rand.Rand) {
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
	fR := float64(rnum.Int63n(backendmessage.GalaxyRadius))
	fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterX = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
	fR = float64(rnum.Int63n(backendmessage.GalaxyRadius))
	fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(randutil.GetPosOrNegOne(rnum))
	fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle
	ss.InterY = int64(fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ)))
}

func generateEllipticalCoordinates(ss *backendmessage.System, rnum *rand.Rand) {
	var X, Y float64
	for {
		ss.InterX = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		ss.InterY = rnum.Int63() * int64(randutil.GetPosOrNegOne(rnum))
		X = float64(ss.InterX * ss.InterX)
		Y = float64(ss.InterY * ss.InterY)
		if math.Sqrt(X+Y) >= backendmessage.GalaxyRadius {
			continue
		}
		break
	}
}
