package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
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

func generateGalaxy(galaxyType int) {

	////////////////////////////////////////////////////////////
	// don't start generating until we have been assigned galaxyName
	////////////////////////////////////////////////////////////

	for {
		switch {
		case len(GLOBAL.backendData.Frontends) == 0:
			time.Sleep(1 * time.Second)
			continue
		case GLOBAL.currentGalaxy.Name == "":
			time.Sleep(1 * time.Second)
			continue
		}
		time.Sleep(1 * time.Second)
		break
	}

	////////////////////////////////////////////////////////////
	// initialize various structures
	////////////////////////////////////////////////////////////

	GLOBAL.currentGalaxy.Solarsystemslice = make([]*SolarSystem, totalNumSystems)
	GLOBAL.systemMutex = make([]sync.RWMutex, totalNumSystems)
	GLOBAL.systemIndex = make(map[string]int, totalNumSystems)

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate systemNames
	////////////////////////////////////////////////////////////

	logInfo(fmt.Sprintf("(backend_init) starting systemName generation (%d threads)",
		GLOBAL.numberOfCores))
	systemsPerCore := totalNumSystems / GLOBAL.numberOfCores
	GLOBAL.waitForCores.Add(GLOBAL.numberOfCores)
	for i := 1; i <= GLOBAL.numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateSystemNames(i, startIndex, stopIndex)
	}
	GLOBAL.waitForCores.Wait()

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate all X,Y coordinates
	////////////////////////////////////////////////////////////

	logInfo(fmt.Sprintf("(backend_init) starting systemCoordinates generation (%d threads)",
		GLOBAL.numberOfCores))
	GLOBAL.waitForCores.Add(GLOBAL.numberOfCores)
	for i := 1; i <= GLOBAL.numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateEllipticalCoordinates(i, startIndex, stopIndex)
	}
	GLOBAL.waitForCores.Wait()

	////////////////////////////////////////////////////////////
	// load coordinates into R* tree data structure (global variable)
	////////////////////////////////////////////////////////////

	logInfo("(backend_init) creating systemCoordinate r-tree datastructure")
	rtreeObjects = make([]rtreego.Spatial, totalNumSystems)
	for i := 0; i < totalNumSystems; i++ {
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[i]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[i] = &SystemBox{rtreego.Point{x, y}, i}
	}
	rtree = rtreego.NewTree(numDimensions, minBranchFactor, maxBranchFactor, rtreeObjects...)

	////////////////////////////////////////////////////////////
	// check (and fix) systems that aren't far enough apart
	////////////////////////////////////////////////////////////

	logInfo(fmt.Sprintf("(backend_init) starting systemCoordinate proximity checks (%d threads)",
		GLOBAL.numberOfCores))
	iter := 0
	for {
		badSystems := [][]int{}
		GLOBAL.waitForCores.Add(GLOBAL.numberOfCores)
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			coreNum := i + 1
			stopIndex := systemsPerCore * coreNum
			startIndex := stopIndex - systemsPerCore
			go func() {
				result := findBadSystemCoordinates(coreNum, startIndex, stopIndex)
				badSystems = append(badSystems, result)
			}()
		}
		GLOBAL.waitForCores.Wait()
		for i := 0; i < GLOBAL.numberOfCores; i++ {
			for x := 0; x < len(badSystems[i]); x++ {
				val := badSystems[i][x]
				generateEllipticalCoordinates(-1, val, val+1)
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
				updateRtreeSystemCoordinates(-1, val, val+1)
			}
		}

		iter++
		logInfo(fmt.Sprint("   ...\tRecreating Rtree\t\tIteration=", iter))
		rtree = rtreego.NewTree(numDimensions, minBranchFactor,
			maxBranchFactor, rtreeObjects...) // !!!
	}

	////////////////////////////////////////////////////////////
	// random sanity checks
	////////////////////////////////////////////////////////////

	logDebug(fmt.Sprintf("There are %v solarSystems", len(GLOBAL.currentGalaxy.GetSolarsystemslice())))
	logDebug(fmt.Sprintf("First Galaxy Name: %s", GLOBAL.currentGalaxy.Solarsystemslice[0].GetName()))
	logDebug(fmt.Sprintf("Random Galaxy Name (9183): %s", GLOBAL.currentGalaxy.Solarsystemslice[9183].Name))
	logDebug(fmt.Sprintf("Random Galaxy Name (123): %s", GLOBAL.currentGalaxy.Solarsystemslice[123].Name))
	logDebug(fmt.Sprintf("Random Galaxy Name (4834): %s", GLOBAL.currentGalaxy.Solarsystemslice[4834].Name))
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

func generateSystemNames(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		GLOBAL.currentGalaxy.Solarsystemslice[currentIndex] = &SolarSystem{}
		sysName := "xyz" + fmt.Sprintf("%v", currentIndex)
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.Name = sysName
	}
	if coreID != -1 {
		GLOBAL.waitForCores.Done()
	}
}

func generateSpiralCoordinates(coreID int, startIndex int, stopIndex int, numArms int) {

	roundFloat := func(val float64) float64 {
		if val < 0 {
			return float64(int64(val - 0.5))
		}
		return float64(int64(val + 0.5))
	}

	posOrNegInt := func() int {
		var randomNum int8
		b := make([]byte, 1)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomNum)
		if randomNum >= 0 {
			return 1
		}
		return -1
	}

	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	rnum := rand.New(seed())

	fDeg2Rad := math.Pi / 180
	fArmAngle := math.Mod(360/float64(numArms), 360)
	fAngularSpread := 90 / float64(numArms)
	fNumArms := float64(numArms)

	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		fK := float64(1)

		fR := float64(rnum.Int63n(galaxyRadius))
		fQ := float64(rnum.Int63n(int64(fAngularSpread))) * float64(posOrNegInt())
		fA := float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle

		interX := fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ))

		fR = float64(rnum.Int63n(galaxyRadius))
		fQ = float64(rnum.Int63n(int64(fAngularSpread))) * float64(posOrNegInt())
		fA = float64(roundFloat(rnum.Float64()*fNumArms)) * fArmAngle

		interY := fR * math.Cos(fDeg2Rad*(fA+(fR*fK)+fQ))

		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		mySystem.InterX = int64(interX)
		mySystem.InterY = int64(interY)
	}

}

func generateEllipticalCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		var interX int64
		var interY int64

		for {
			b := make([]byte, 8)
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
		GLOBAL.waitForCores.Done()
	}
}

func findBadSystemCoordinates(coreID int, startIndex int, stopIndex int) (myBadSystems []int) {
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
		GLOBAL.waitForCores.Done()
	}
	return myBadSystems
}

func updateRtreeSystemCoordinates(coreID int, startIndex int, stopIndex int) {
	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {
		//rtree.Delete(rtreeObjects[currentIndex]) // !!! TODO: NEED TO FIX
		mySystem := GLOBAL.currentGalaxy.Solarsystemslice[currentIndex]
		x := float64(mySystem.InterX)
		y := float64(mySystem.InterY)
		rtreeObjects[currentIndex] = &SystemBox{rtreego.Point{x, y}, currentIndex}
		//rtree.Insert(rtreeObjects[currentIndex]) // !!! TODO: NEED TO FIX
	}
	if coreID != -1 {
		GLOBAL.waitForCores.Done()
	}
}
