package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/hyperionfive/names"
	"github.com/demskie/randutil"
	"github.com/demskie/spatialindex"

	"github.com/gogo/protobuf/proto"
)

// UnmarshalFile ...
func (gal *GalaxyStruct) UnmarshalFile() (rt *spatialindex.Tree) {
	const fName = "(galaxy.UnmarshalFile)"
	// after deserialization we need to call the following function
	createTree := func() {
		for _, ss := range gal.Protobuf.Systemslice {
			x := float64(ss.InterX)
			y := float64(ss.InterY)
			gal.Rtree.Add(ss.NameUint, x, y)
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
	gal.Protobuf.Systemslice = make([]*gamestate.System, totalNumSystems)
	for i := range gal.Protobuf.Systemslice {
		gal.Protobuf.Systemslice[i] = &gamestate.System{}
	}
	gal.Protobuf.Planetmap = make(map[uint64]*gamestate.Planet)
	gal.Protobuf.Objectmap = make(map[uint64]*gamestate.Object)
	gal.Protobuf.Systemmap = make(map[uint64]*gamestate.System)
	gal.Protobuf.Fleetmap = make(map[int32]*gamestate.Fleet)
	gal.Protobuf.Battlemap = make(map[int32]*gamestate.Battle)
	gal.Protobuf.Siegemap = make(map[int32]*gamestate.Siege)
	gal.Protobuf.Actionmap = make(map[int32]*gamestate.Action)
	localMutex := sync.Mutex{}
	badSystems := int32(0)
	log.Info(fName, fmt.Sprintf("generating %v systems",
		intutil.CommaFormatNum(len(gal.Protobuf.Systemslice))))
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartStopForCore(
			coreNum, args.NumberOfCores, len(gal.Protobuf.Systemslice))
		if stopIndex <= startIndex {
			return
		}
		var ss *gamestate.System
		var x1, y1, x2, y2 float64
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.Protobuf.Systemslice[i]
			generateSystem(ss, rnum, gal.SystemAndPlanetNames)
			for {
				generateEllipticalCoordinates(ss, rnum)
				x1, y1 = float64(ss.InterX), float64(ss.InterY)
				gal.Rtree.Add(ss.NameUint, x1, y1)
				objects, err := gal.Rtree.NearestNeighbors(ss.NameUint, 1)
				if err != spatialindex.ErrNotEnoughNeighbors &&
					log.CheckError(err, fName) {
					log.SuddenExit()
				}
				if len(objects) == 0 {
					continue
				}
				x2, y2 = objects[0].X, objects[0].Y
				if math.Hypot(x1-x2, y1-y2) <= gamestate.MinimumDistanceBetweenSystems {
					gal.Rtree.Delete(ss.NameUint)
					atomic.AddInt32(&badSystems, 1)
					continue
				}
				break
			}
			localMutex.Lock()
			for _, pl := range ss.Planetslice {
				gal.Protobuf.Planetmap[pl.NameUint] = pl
			}
			for _, ob := range ss.Objectslice {
				gal.Protobuf.Objectmap[ob.NameUint] = ob
			}
			gal.Protobuf.Systemmap[ss.NameUint] = ss
			localMutex.Unlock()
		}
	})
	log.Info(fName, fmt.Sprintf("regenerated %v systems", intutil.CommaFormatNum(badSystems)))
	log.Info(fName, "generating system links")
	gal.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartStopForCore(
			coreNum, args.NumberOfCores, len(gal.Protobuf.Systemslice))
		if stopIndex <= startIndex {
			return
		}
		var ss *gamestate.System
		var normalDistVal int
		rnum := gal.ProcessingRnumSlice[coreNum]
		for i := startIndex; i < stopIndex; i++ {
			ss = gal.Protobuf.Systemslice[i]
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			normalDistVal = 4 + int(rnum.NormFloat64()*1.2)
			if normalDistVal < 1 {
				normalDistVal = 1
			} else if normalDistVal > 9 {
				normalDistVal = 9
			}
			ss.ConnectedSystemsUint = make([]uint64, normalDistVal)
			neighbors, err := gal.Rtree.NearestNeighbors(ss.NameUint, normalDistVal)
			if err != spatialindex.ErrNotEnoughNeighbors &&
				log.CheckError(err, fName) {
				log.SuddenExit()
			}
			for i, neighbor := range neighbors {
				ss.ConnectedSystemsUint[i] = neighbor.ID
			}
		}
	})
}

func generateSystem(ss *gamestate.System, rnum *rand.Rand, nameGen *names.NameGenerator) {
	const fName = "(generateSystem)"

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
	var rawName *names.NameStruct
	rawName = nameGen.GenerateName(rnum)
	ss.NameUint = rawName.GetNameUint()
	nameGen.AddToIndex(rawName)

	// generate number of planets
	var numPlanets int64
	if ss.SystemType == gamestate.YELLOWGIANT {
		numPlanets = int64(math.Abs(rnum.NormFloat64()*7) + 20)
	} else if ss.SystemType == gamestate.REDGIANT {
		numPlanets = int64(math.Abs(rnum.NormFloat64()*3) + 25)
	} else if ss.SystemType == gamestate.BLUEGIANT {
		numPlanets = int64(math.Abs(rnum.NormFloat64()*7) + 20)
	} else if ss.SystemType == gamestate.NEUTRONSTAR {
		numPlanets = int64(math.Abs(rnum.NormFloat64()*7) + 10)
	} else if ss.SystemType == gamestate.WHITEDWARF {
		numPlanets = int64(0)
	} else if ss.SystemType == gamestate.BLACKHOLE {
		numPlanets = int64(rnum.NormFloat64()*10 + 30)
		if numPlanets < int64(11) {
			numPlanets = int64(11)
		} else if numPlanets > int64(72) {
			numPlanets = int64(52)
		}
	} else {
		log.Error(fName, fmt.Sprintf("invalid system.Type: %v", ss.SystemType))
		log.SuddenExit()
	}
	if numPlanets > int64(72) {
		numPlanets = int64(53)
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

	// start building planets
	ss.Planetslice = []*gamestate.Planet{}
	for i := 0; i < int(numPlanets); i++ {
		newPlanet := &gamestate.Planet{}
		// generate type
		result, err := randutil.WeightedChoice(weightedObjTypes, rnum)
		log.CheckError(err, fName)
		newPlanet.PlanetType = result.Object.(int32)
		// generate name
		rawName = nameGen.GenerateName(rnum)
		newPlanet.NameUint = rawName.GetNameUint()
		nameGen.AddToIndex(rawName)
		// insert parentSystemName
		newPlanet.ParentSystemUint = ss.NameUint
		// append result
		ss.Planetslice = append(ss.Planetslice, newPlanet)
	}

	// TODO // build objects
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
