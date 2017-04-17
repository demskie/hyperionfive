package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

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
)

var (
	waitForGen  sync.WaitGroup
	systemMutex []sync.RWMutex
	systemIndex map[string]int
)

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

	waitForGen.Add(numberOfCores)
	for i := 1; i <= numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateSystemNames(i, startIndex, stopIndex)
	}
	waitForGen.Wait()

	////////////////////////////////////////////////////////////
	// spawn goroutines to generate all X,Y coordinates
	////////////////////////////////////////////////////////////

	waitForGen.Add(numberOfCores)
	for i := 1; i <= numberOfCores; i++ {
		stopIndex := systemsPerCore * i
		startIndex := stopIndex - systemsPerCore
		go generateSystemCoordinates(i, startIndex, stopIndex)
	}
	waitForGen.Wait()

	//fmt.Printf("\nThere are %v solarSystems\n", len(currentGalaxy.GetSolarsystemslice()))

	//fmt.Printf("\nFirst Galaxy Name: %s", currentGalaxy.Solarsystemslice[0].GetName())
	//fmt.Printf("\nRandom Galaxy Name: %s", currentGalaxy.Solarsystemslice[9183].Name)

	//os.Exit(1)

	////////////////////////////////////////////////////////////
	// ensure coordinates meet constraints (singlethreaded)
	////////////////////////////////////////////////////////////

	//fixSystemCoordinates()

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

	waitForGen.Done()
}

func generateSystemCoordinates(coreID int, startIndex int, stopIndex int) {

	for currentIndex := startIndex; currentIndex < stopIndex; currentIndex++ {

		if currentIndex%maxNumberOfPlayers == 0 {
			logInfo(fmt.Sprintf("(backend_init) Thread %d is %.2f%% complete generating system coordinates",
				coreID, float64(currentIndex-startIndex)/float64(stopIndex)))
		}

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
			if checkSystemCoordinates(currentIndex, interX, interY) == false {
				continue // try again
			}
			break
		}

		// assigning coordinates and name
		mySystem := currentGalaxy.Solarsystemslice[currentIndex]
		systemMutex[currentIndex].Lock()
		mySystem.InterX = interX
		mySystem.InterY = interY
		systemMutex[currentIndex].Unlock()
		//v := reflect.ValueOf(tmp)
		//fmt.Printf("Kind: %v\t Type: %v\t\tNumFields: %v\n",
		//	v.Kind(), v.Type(), v.NumField())

	}

	// decrementing waitgroup and printing completion status
	logInfo(fmt.Sprintf("(backend_init) Thread %d has finished generating system coordinates",
		coreID))

	waitForGen.Done()
}

func checkSystemCoordinates(myIndex int, myInterX int64, myInterY int64) bool {
	for anotherIndex := 0; anotherIndex < totalNumSystems; anotherIndex++ {
		if myIndex >= anotherIndex {
			anotherIndex = myIndex + 1
			continue
		}
		//logInfo(fmt.Sprintf("anotherIndex: %v", anotherIndex))
		anotherSystem := currentGalaxy.Solarsystemslice[anotherIndex]
		systemMutex[anotherIndex].RLock()
		anotherInterX := anotherSystem.InterX
		anotherInterY := anotherSystem.InterY
		systemMutex[anotherIndex].RUnlock()

		if anotherInterX == 0 && anotherInterY == 0 {
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
