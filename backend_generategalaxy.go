package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/dhconnelly/rtreego"

	cryptorand "crypto/rand"
)

var (
	rtree        *rtreego.Rtree
	rtreeObjects []rtreego.Spatial
	badSystems   [][]int
)

// SystemBox ...
type SystemBox struct {
	location rtreego.Point
	index    int
}

// Bounds ...
func (sb *SystemBox) Bounds() *rtreego.Rect {
	tol := 0.01
	return sb.location.ToRect(tol)
}

func generateGalaxy(galaxyType int32) {
	fName := "(generateGalaxy)"

	////////////////////////////////////////////////////////////
	// don't start generating until we have been assigned galaxyName
	////////////////////////////////////////////////////////////

	if false {
		for {
			switch {
			case len(GLOBAL.backendData.Frontends) == 0:
				logError(fName, "waiting for fronendSync")
				time.Sleep(1 * time.Second)
				continue
			case GLOBAL.currentGalaxy.Name == "":
				logError(fName, "waiting to be assigned a galaxyName")
				time.Sleep(1 * time.Second)
				continue
			}
			time.Sleep(1 * time.Second)
			break
		}
	}

	var totalNumSystems int
	if GLOBAL.debugModeBool != true {
		totalNumSystems = maxNumberOfPlayers * numSystemsPerPlayer
	} else {
		totalNumSystems = 10000
	}

	////////////////////////////////////////////////////////////
	// initialize various structures
	////////////////////////////////////////////////////////////

	wg := sync.WaitGroup{}
	GLOBAL.currentGalaxy.Solarsystemslice = make([]*SolarSystem, totalNumSystems)

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate systems
	////////////////////////////////////////////////////////////

	logInfo(fName, "totalNumSystems "+commaFormatUint(uint64(totalNumSystems)))
	logInfo(fName, fmt.Sprintf("starting basic system generation (%d threads)",
		GLOBAL.numberOfCores))
	systemsPerCore := totalNumSystems / GLOBAL.numberOfCores
	wg.Add(GLOBAL.numberOfCores)
	for i := 1; i <= GLOBAL.numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		coreID := i
		go generateSystems(coreID, startIndex, stopIndex, &wg)
	}
	wg.Wait()

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate all X,Y coordinates for systems
	////////////////////////////////////////////////////////////

	logInfo(fName, fmt.Sprintf("starting systemCoordinates generation (%d threads)",
		GLOBAL.numberOfCores))
	wg.Add(GLOBAL.numberOfCores)
	for i := 1; i <= GLOBAL.numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		coreID := i
		go generateEllipticalCoordinates(coreID, startIndex, stopIndex, &wg)
	}
	wg.Wait()

	////////////////////////////////////////////////////////////
	// load coordinates into R* tree data structure (global variable)
	////////////////////////////////////////////////////////////

	logInfo(fName, "creating systemCoordinate r-tree datastructure")
	rtreeObjects = make([]rtreego.Spatial, totalNumSystems)
	for i := 0; i < totalNumSystems; i++ {
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[i]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[i] = &SystemBox{rtreego.Point{x, y}, i}
	}
	rtree = rtreego.NewTree(numDimensions, minBranchFactor, maxBranchFactor, rtreeObjects...)

	////////////////////////////////////////////////////////////
	// check (and fix) system coordinates that aren't far enough apart
	////////////////////////////////////////////////////////////

	logInfo(fName, fmt.Sprintf("starting systemCoordinate proximity checks (%d threads)",
		GLOBAL.numberOfCores))
	iter := 0
	for {
		badSystems := [][]int{}
		wg.Add(GLOBAL.numberOfCores)
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := systemsPerCore * coreNum
			startIndex := stopIndex - systemsPerCore
			go func() {
				result := findBadSystemCoordinates(coreNum, startIndex, stopIndex, &wg)
				badSystems = append(badSystems, result)
			}()
		}
		wg.Wait()
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			for x := 0; x < len(badSystems[i]); x++ {
				val := badSystems[i][x]
				generateEllipticalCoordinates(-1, val, val+1, &wg)
			}
		}

		successful := true
		for i := 0; i < len(badSystems); i++ {
			if len(badSystems[i]) > 0 {
				successful = false
				logInfo(fmt.Sprintf("   ...\t\tThread=%v\t\t\tNumberOfBadSystems=%v",
					i, len(badSystems[i])))
			}
		}
		if successful {
			break
		}

		for i := 0; i < len(badSystems); i++ {
			for x := 0; x < len(badSystems[i]); x++ {
				val := badSystems[i][x]
				updateRtreeSystemCoordinates(-1, val, val+1, &wg)
			}
		}

		iter++
		logInfo(fmt.Sprint("   ...\tRecreating Rtree\t\tIteration=", iter))
		rtree = rtreego.NewTree(numDimensions, minBranchFactor,
			maxBranchFactor, rtreeObjects...) // !!!

	}

	////////////////////////////////////////////////////////////
	// check (and fix) system names that are too similar
	////////////////////////////////////////////////////////////

	// TODO // implement logic

	////////////////////////////////////////////////////////////
	// random sanity checks
	////////////////////////////////////////////////////////////

	logDebug(fmt.Sprintf("First System Name: %s", GLOBAL.currentGalaxy.Solarsystemslice[0].GetName()))
	logDebug(fmt.Sprintf("Random System Name (9183): %s", GLOBAL.currentGalaxy.Solarsystemslice[9183].Name))
	logDebug(fmt.Sprintf("Random System Name (123): %s", GLOBAL.currentGalaxy.Solarsystemslice[123].Name))
	logDebug(fmt.Sprintf("Random System Name (4834): %s", GLOBAL.currentGalaxy.Solarsystemslice[4834].Name))

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSystems(coreID int, startIndex int, stopIndex int, wg *sync.WaitGroup) {
	fName := "(generateSystems)"

	// initialize random seed
	rnum := getCryptoRnum()

	// define weight for systemTypes
	weightedSystemTypes := []weightedObj{
		weightedObj{Weight: 50, Item: YELLOWGIANT},
		weightedObj{Weight: 50, Item: REDGIANT},
		weightedObj{Weight: 50, Item: BLUEGIANT},
		weightedObj{Weight: 10, Item: NEUTRONSTAR},
		weightedObj{Weight: 10, Item: WHITEDWARF},
		weightedObj{Weight: 5, Item: BLACKHOLE},
	}

	// preemptively load all txt files containing names
	temp := NamedObject{}
	temp.LoadAllNames()

	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {

		// initialize system and create a short reference
		GLOBAL.currentGalaxy.Solarsystemslice[currentIndex] = &SolarSystem{}
		system := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]

		// generate type
		result, err := weightedChoice(weightedSystemTypes, rnum)
		checkError(err, fName)
		system.Type = result.Item.(int32)

		// generate name
		canidateName := generateName(SOLARSYSTEMNAME, system.Type)
		system.Name = canidateName.GetFullName()

		// generate number of objects
		var numObj int64
		if system.Type == YELLOWGIANT {
			numObj = int64(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if system.Type == REDGIANT {
			numObj = int64(math.Abs(rnum.NormFloat64()*3) + 25)
		} else if system.Type == BLUEGIANT {
			numObj = int64(math.Abs(rnum.NormFloat64()*7) + 20)
		} else if system.Type == NEUTRONSTAR {
			numObj = int64(math.Abs(rnum.NormFloat64()*7) + 10)
		} else if system.Type == WHITEDWARF {
			numObj = int64(0)
		} else if system.Type == BLACKHOLE {
			numObj = int64(rnum.NormFloat64()*10 + 30)
			if numObj < int64(11) {
				numObj = int64(11)
			} else if numObj > int64(72) {
				numObj = int64(52)
			}
		} else {
			logError(fName, "invalid system.Type:", fmt.Sprint(system.Type))
			suddenExit()
		}
		if numObj > int64(72) {
			numObj = int64(53)
		}
		system.Solarobjectslice = make([]*SolarObject, numObj)

		// define weight for this object based on SolarSystemType
		var weightedObjTypes []weightedObj
		if system.Type == YELLOWGIANT {
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
		} else if system.Type == REDGIANT {
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
		} else if system.Type == BLUEGIANT {
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
		} else if system.Type == NEUTRONSTAR {
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
		} else if system.Type == WHITEDWARF {
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
		} else if system.Type == BLACKHOLE {
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

		// start building objects in the preallocated array
		for i := 0; i < len(system.Solarobjectslice); i++ {
			system.Solarobjectslice[i] = &SolarObject{}
			obj := system.Solarobjectslice[i]

			// generate type
			result, err := weightedChoice(weightedObjTypes, rnum)
			checkError(err, fName)
			obj.Type = result.Item.(int32)

			// generate name
			canidateSystemName := generateName(SOLARSYSTEMNAME, obj.Type)
			obj.Name = canidateSystemName.GetFullName()
		}
	}
	if coreID != -1 {
		wg.Done()
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSpiralCoordinates(coreID int, startIndex int, stopIndex int, numArms int) {
	roundFloat := func(val float64) float64 {
		if val < 0 {
			return float64(int64(val - 0.5))
		}
		return float64(int64(val + 0.5))
	}
	rnum := getCryptoRnum()

	fDeg2Rad := math.Pi / 180
	fArmAngle := math.Mod(360/float64(numArms), 360)
	fAngularSpread := 90 / float64(numArms)
	fNumArms := float64(numArms)

	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		fK := float64(1)

		fR := float64(rnum.Int63n(galaxyRadius))
		fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(getPosOrNegOne())
		fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle

		interX := fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ))

		fR = float64(rnum.Int63n(galaxyRadius))
		fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(getPosOrNegOne())
		fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle

		interY := fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ))

		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.InterX = int64(interX)
		mySystem.InterY = int64(interY)
	}

}

func generateEllipticalCoordinates(coreID int, startIndex int, stopIndex int, wg *sync.WaitGroup) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		var interX int64
		var interY int64

		b := make([]byte, 8)
		for {
			cryptorand.Read(b)
			buf := bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interX)

			cryptorand.Read(b)
			buf = bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interY)

			A2 := float64(interX * interX)
			B2 := float64(interY * interY)
			C := math.Sqrt(A2 + B2)
			if C >= galaxyRadius {
				continue
			}

			break
		}
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.InterX = interX
		mySystem.InterY = interY
	}

	if coreID != -1 {
		wg.Done()
	}
}

func findBadSystemCoordinates(coreID int, startIndex int, stopIndex int, wg *sync.WaitGroup) (myBadSystems []int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]

		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)

		nearestNeighbors := rtree.NearestNeighbors(2, rtreego.Point{x, y})

		var validNeighborX int64
		var validNeighborY int64
		var distanceBetweenSystems int64

		if nearestNeighbors[0].(*SystemBox).index == currentIndex {
			validNeighborX = int64(nearestNeighbors[1].Bounds().PointCoord(0))
			validNeighborY = int64(nearestNeighbors[1].Bounds().PointCoord(1))
		} else {
			validNeighborX = int64(nearestNeighbors[0].Bounds().PointCoord(0))
			validNeighborY = int64(nearestNeighbors[0].Bounds().PointCoord(1))
		}

		value := mySystem.InterX - validNeighborX
		if value < 0 {
			distanceBetweenSystems = -value
		} else {
			distanceBetweenSystems = value
		}
		if distanceBetweenSystems <= minimumDistanceBetweenSystems {
			myBadSystems = append(myBadSystems, currentIndex)
			//logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
			//	mySystem.Name,
			//	mySystem.InterX,
			//	validNeighborX,
			//	distanceBetweenSystems))
			continue
		}

		value = mySystem.InterY - validNeighborY
		if value < 0 {
			distanceBetweenSystems = -value
		} else {
			distanceBetweenSystems = value
		}
		if distanceBetweenSystems <= minimumDistanceBetweenSystems {
			myBadSystems = append(myBadSystems, currentIndex)
			//logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
			//	mySystem.Name,
			//	mySystem.InterY,
			//	validNeighborY,
			//	distanceBetweenSystems))
			continue
		}
	}
	if coreID != -1 {
		wg.Done()
	}
	return myBadSystems
}

func updateRtreeSystemCoordinates(coreID int, startIndex int, stopIndex int, wg *sync.WaitGroup) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		//rtree.Delete(rtreeObjects[currentIndex]) // !!! TODO: NEED TO FIX
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[currentIndex] = &SystemBox{rtreego.Point{x, y}, currentIndex}
		//rtree.Insert(rtreeObjects[currentIndex]) // !!! TODO: NEED TO FIX
	}
	if coreID != -1 {
		wg.Done()
	}
}
