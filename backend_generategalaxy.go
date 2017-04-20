package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/dhconnelly/rtreego"

	cryptorand "crypto/rand"
)

const (
	lightyear                     = 1000000 // one million
	maxNumberOfPlayers            = 10000
	numSystemsPerPlayer           = 100
	totalNumSystems               = maxNumberOfPlayers * numSystemsPerPlayer
	maxDistanceFromCenter         = math.MaxInt64
	minimumDistanceBetweenObjects = 1000
	minimumDistanceBetweenSystems = 1000
	oneMillion                    = 1000000
	oneBillion                    = 1000000000
)

var (
	waitForCores sync.WaitGroup
	systemMutex  []sync.RWMutex
	systemIndex  map[string]int
)

var (
	rtree        *rtreego.Rtree
	rtreeObjects []rtreego.Spatial
)

const (
	numDimensions   = 2
	minBranchFactor = 5
	maxBranchFactor = 25
)

// galaxytype
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateGalaxy() {

	currentGalaxy.Name = "Andromeda-187"

	////////////////////////////////////////////////////////////
	// initialize various structures
	////////////////////////////////////////////////////////////

	currentGalaxy.Solarsystemslice = make([]*SolarSystem, totalNumSystems)
	systemMutex = make([]sync.RWMutex, totalNumSystems)
	systemIndex = make(map[string]int, totalNumSystems)

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate systemNames
	////////////////////////////////////////////////////////////

	systemsPerCore := totalNumSystems / numberOfCores

	waitForCores.Add(numberOfCores)
	for i := 1; i <= numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateSystemNames(i, startIndex, stopIndex)
	}
	waitForCores.Wait()

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate all X,Y coordinates
	////////////////////////////////////////////////////////////

	waitForCores.Add(numberOfCores)
	for i := 1; i <= numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateSystemCoordinates(i, startIndex, stopIndex)
	}
	waitForCores.Wait()

	////////////////////////////////////////////////////////////
	// load coordinates into R* tree data structure (global variable)
	////////////////////////////////////////////////////////////

	rtreeObjects = make([]rtreego.Spatial, totalNumSystems)
	for i := 0; i < totalNumSystems; i++ {
		mySystem := currentGalaxy.Solarsystemslice[i]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[i] = &SystemBox{rtreego.Point{x, y}, i}
	}
	rtree = rtreego.NewTree(numDimensions, minBranchFactor, maxBranchFactor, rtreeObjects...)

	////////////////////////////////////////////////////////////
	// check (and fix) systems that aren't far enough apart
	////////////////////////////////////////////////////////////

	waitForCores.Add(numberOfCores)
	for i := 1; i <= numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go func() {
			for {
				myBadSystems := findBadSystemCoordinates(-1, startIndex, stopIndex)
				if len(myBadSystems) == 0 {
					break
				}
				for idx := range myBadSystems {
					systemMutex[idx].Lock()
					generateSystemCoordinates(-1, idx, idx+1)
					updateRtreeSystemCoordinates(-1, idx, idx+1)
					systemMutex[idx].Unlock()
				}
			}
			waitForCores.Done()
		}()
	}
	waitForCores.Wait()

}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSystemNames(coreID int, startIndex int, stopIndex int) {

	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		// initialize solarSystem with default values
		currentGalaxy.Solarsystemslice[currentIndex] = &SolarSystem{}

		// create system name based on index number (need to refactor)
		sysName := "xyz" + fmt.Sprintf("%v", currentIndex)

		// assign name to this system
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.Name = sysName
	}

	// decrementing waitgroup and printing completion status
	logInfo(fmt.Sprintf("(backend_init) Thread %d has finished generating system names",
		coreID))

	if coreID != -1 {
		waitForCores.Done()
	}
}

func generateSystemCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		// not necessary at the moment
		//if currentIndex%maxNumberOfPlayers == 0 && coreID != -1 {
		//	logInfo(fmt.Sprintf("(backend_init) Thread %d is %.2f%% complete generating"+
		//		"system coordinates", coreID, float64(currentIndex-startIndex)/float64(stopIndex)))
		//}

		// generating a single pair of x and y coordinates
		var interX int64
		var interY int64
		for {
			// generating a uniformly random int64 value
			b := make([]byte, 8)
			cryptorand.Read(b)
			buf := bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interX)

			// generating another uniformly random int64 value
			cryptorand.Read(b)
			buf = bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interY)

			// calculate hypotenuse and throw away any
			// coordinates that would fall outside the circle
			A2 := float64(interX * interX)
			B2 := float64(interY * interY)
			C := math.Sqrt(A2 + B2)
			if C >= maxDistanceFromCenter {
				continue // try again
			}
			break
		}

		// assigning coordinates and name
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.InterX = interX
		mySystem.InterY = interY
		//v := reflect.ValueOf(tmp)
		//fmt.Printf("Kind: %v\t Type: %v\t\tNumFields: %v\n",
		//	v.Kind(), v.Type(), v.NumField())

	}

	// decrementing waitgroup and printing completion status
	logInfo(fmt.Sprintf("(backend_init) Thread %d has finished generating system coordinates",
		coreID))

	if coreID != -1 {
		waitForCores.Done()
	}
}

func findBadSystemCoordinates(coreID int, startIndex int, stopIndex int) (badSystems []int) {
	badSystems = []int{}
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
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
			badSystems = append(badSystems, currentIndex)
			logInfo(fmt.Sprintf("(backend_init) Detected bad system coordinates for %s",
				mySystem.Name))
			logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
				mySystem.Name,
				mySystem.InterX,
				validNeighborX,
				value))
		}

		value = mySystem.InterY - validNeighborY
		if value < 0 {
			distanceBetweenSystems = -value
		} else {
			distanceBetweenSystems = value
		}
		if distanceBetweenSystems <= minimumDistanceBetweenSystems {
			logInfo(fmt.Sprintf("(backend_init) Detected bad system coordinates for %s",
				mySystem.Name))
			logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
				mySystem.Name,
				mySystem.InterY,
				validNeighborY,
				value))
		}
	}

	if coreID != -1 {
		waitForCores.Done()
	}
	return badSystems
}

func updateRtreeSystemCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		if rtree.Delete(rtreeObjects[currentIndex]) != true {
			logInfo("(backend_init) Rtree failed to delete a record for an unknown reason")
		}
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtree.Insert(&SystemBox{rtreego.Point{x, y}, currentIndex})
	}
	if coreID != -1 {
		waitForCores.Done()
	}
}

func slowlyCheckSystemCoordinates(myIndex int, myInterX int64, myInterY int64) bool {
	for anotherIndex := 0; anotherIndex < totalNumSystems; anotherIndex++ {
		if myIndex >= anotherIndex {
			anotherIndex = myIndex + 1
			continue
		}
		anotherSystem := currentGalaxy.Solarsystemslice[anotherIndex]
		systemMutex[anotherIndex].RLock()
		anotherInterX := anotherSystem.InterX
		anotherInterY := anotherSystem.InterY
		systemMutex[anotherIndex].RUnlock()

		if anotherInterX == 0 && anotherInterY == 0 {
			// default value (0,0) is not valid, must skip
			continue
		}

		var distanceBetweenSystems int64

		value := myInterX - anotherInterX
		if value < 0 {
			distanceBetweenSystems = -value
		} else {
			distanceBetweenSystems = value
		}
		if distanceBetweenSystems <= minimumDistanceBetweenSystems {
			return false
			//logInfo(fmt.Sprintf("%v", value))
			//logInfo(fmt.Sprintf("%v", distanceBetweenSystems))
			//logInfo(fmt.Sprintf("myInterX: %v anotherInterX: %v\n",
			//	myInterX, anotherInterX))
		}

		value = myInterY - anotherInterY
		if value < 0 {
			distanceBetweenSystems = -value
		} else {
			distanceBetweenSystems = value
		}
		if distanceBetweenSystems <= minimumDistanceBetweenSystems {
			return false
			//logInfo(fmt.Sprintf("%v", value))
			//logInfo(fmt.Sprintf("%v", distanceBetweenSystems))
			//logInfo(fmt.Sprintf("myInterY: %v anotherInterY: %v\n",
			//	myInterY, anotherInterY))
		}
	}
	return true
}
