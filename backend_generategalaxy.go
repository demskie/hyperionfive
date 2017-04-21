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
	maxNumberOfPlayers            = 10000
	numSystemsPerPlayer           = 100
	totalNumSystems               = maxNumberOfPlayers * numSystemsPerPlayer
	maxDistanceFromCenter         = math.MaxInt64 // lightyears
	lightyear                     = fakeToRealMultiplier
	minimumDistanceBetweenSystems = lightyear
	fakeGalaxyRadius              = math.MaxInt64 // lightyears
	realGalaxyRadius              = 125000        // lightyears
	fakeToRealMultiplier          = fakeGalaxyRadius / realGalaxyRadius
	oneMillion                    = 1000000
	oneBillion                    = 1000000000
	oneTrillion                   = 1000000000000
)

var (
	waitForCores sync.WaitGroup
	systemMutex  []sync.RWMutex
	systemIndex  map[string]int
)

var (
	rtree        *rtreego.Rtree
	rtreeObjects []rtreego.Spatial
	badSystems   [][]int
)

const (
	numDimensions   = 2
	minBranchFactor = 1
	maxBranchFactor = 2
)

// galaxytype
const (
	ELLIPTICAL int = iota
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

func generateGalaxy(galaxyType int) {

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

	logInfo(fmt.Sprintf("(backend_init) starting systemName generation (%d threads)",
		numberOfCores))
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

	logInfo(fmt.Sprintf("(backend_init) starting systemCoordinates generation (%d threads)",
		numberOfCores))
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

	logInfo("(backend_init) creating systemCoordinate r-tree datastructure")
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

	logInfo(fmt.Sprintf("(backend_init) starting systemCoordinate proximity checks (%d threads)",
		numberOfCores))
	iter := 0
	for {
		badSystems := [][]int{}
		waitForCores.Add(numberOfCores)
		for i := 0; i < numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := systemsPerCore * coreNum
			startIndex := stopIndex - systemsPerCore
			go func() {
				result := findBadSystemCoordinates(coreNum, startIndex, stopIndex)
				badSystems = append(badSystems, result)
			}()
		}
		waitForCores.Wait()
		for i := 0; i < numberOfCores; i++ {
			for x := 0; x < len(badSystems[i]); x++ {
				val := badSystems[i][x]
				generateSystemCoordinates(-1, val, val+1)
			}
		}

		successful := true
		//logInfo(fmt.Sprint(badSystems))
		//logInfo(fmt.Sprint(len(badSystems)))
		for i := 0; i < len(badSystems); i++ {
			if len(badSystems[i]) > 0 {
				successful = false
				logInfo(fmt.Sprintf("   ...\t\tThread=%v\t\t\tNumberOfBadSystems=%v", i, len(badSystems[i])))
			}
		}
		if successful {
			break
		}

		for i := 0; i < len(badSystems); i++ {
			for x := 0; x < len(badSystems[i]); x++ {
				val := badSystems[i][x]
				updateRtreeSystemCoordinates(-1, val, val+1)
			}
		}

		iter++
		logInfo(fmt.Sprint("   ...\tRecreating Rtree\t\tIteration=", iter))
		rtree = rtreego.NewTree(numDimensions, minBranchFactor, maxBranchFactor, rtreeObjects...) // !!!
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSystemNames(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		/////////////////////////////////////////////
		// initialize solarSystem with default values
		/////////////////////////////////////////////
		currentGalaxy.Solarsystemslice[currentIndex] = &SolarSystem{}

		//////////////////////////////////////////////////////////////
		// create system name based on index number (need to refactor)
		/////////////////////////////////////////////////////////////
		sysName := "xyz" + fmt.Sprintf("%v", currentIndex)

		/////////////////////////////
		// assign name to this system
		/////////////////////////////
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.Name = sysName
	}

	////////////////////////////////////////////////////////
	// decrementing waitgroup
	////////////////////////////////////////////////////////
	if coreID != -1 {
		waitForCores.Done()
	}
}

func generateSpiralGalaxy(coreID int, startIndex int, stopIndex int, numSpirals int) {

}

func generateSystemCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		var interX int64
		var interY int64

		for {
			////////////////////////////////////////////
			// generating a uniformly random int64 value
			////////////////////////////////////////////
			b := make([]byte, 8)
			cryptorand.Read(b)
			buf := bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interX)

			//////////////////////////////////////////////////
			// generating another uniformly random int64 value
			//////////////////////////////////////////////////
			cryptorand.Read(b)
			buf = bytes.NewBuffer(b)
			binary.Read(buf, binary.LittleEndian, &interY)

			/////////////////////////////////////////////////
			// calculate hypotenuse and throw away any
			// coordinates that would fall outside the circle
			/////////////////////////////////////////////////
			A2 := float64(interX * interX)
			B2 := float64(interY * interY)
			C := math.Sqrt(A2 + B2)
			if C >= maxDistanceFromCenter {
				continue
			}

			break
		}

		/////////////////////////////////
		// assigning coordinates and name
		/////////////////////////////////
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.InterX = interX
		mySystem.InterY = interY

	}

	////////////////////////////////////////////////////////
	// decrementing waitgroup and printing completion status
	////////////////////////////////////////////////////////
	if coreID != -1 {
		waitForCores.Done()
	}
}

func findBadSystemCoordinates(coreID int, startIndex int, stopIndex int) (myBadSystems []int) {
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
			myBadSystems = append(myBadSystems, currentIndex)
			logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
				mySystem.Name,
				mySystem.InterX,
				validNeighborX,
				distanceBetweenSystems))
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
			logDebug(fmt.Sprintf("%s\t%v\t%v\t%v",
				mySystem.Name,
				mySystem.InterY,
				validNeighborY,
				distanceBetweenSystems))
			continue
		}
	}
	if coreID != -1 {
		waitForCores.Done()
	}
	return myBadSystems
}

func updateRtreeSystemCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		//rtree.Delete(rtreeObjects[currentIndex]) // !!!

		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[currentIndex] = &SystemBox{rtreego.Point{x, y}, currentIndex}

		//rtree.Insert(rtreeObjects[currentIndex]) // !!!
	}
	if coreID != -1 {
		waitForCores.Done()
	}
}
