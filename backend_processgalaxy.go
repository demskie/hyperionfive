package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
)

func processGalaxy(futureGalaxy *Galaxy, sysIndex map[string]int) {
	const fName = "(processGalaxy)"
	start := time.Now()

	if GLOBAL.galaxy.Solarsystemslice != nil {
		deepCopyGalaxy(GLOBAL.galaxy, futureGalaxy, sysIndex)
	} else {
		logError(fName, "just tried to process a nil Galaxy")
		return
	}

	// estimate futureGalaxy tick time by referencing previous tick time
	GLOBAL.backendData.Lock() // BUG // not sure if this is needed
	longestTickTime := GLOBAL.backendData.LocalTickTime
	for _, neighbor := range GLOBAL.backendData.ReceivedMessageFrom {
		remoteTickTime := GLOBAL.backendData.NeighborTickTimes[neighbor]
		if remoteTickTime > longestTickTime {
			longestTickTime = remoteTickTime
		}
	}
	timeMultiplier := longestTickTime.Seconds()
	if timeMultiplier <= float64(0) {
		timeMultiplier = float64(1)
	}
	rnumMapping := make([]*rand.Rand, GLOBAL.numberOfCores)
	for coreID := range rnumMapping {
		seed := int64(coreID)
		for _, sysHash := range GLOBAL.backendData.SystemHashes {
			seed += sysHash
		}
		rnumMapping[coreID] = rand.New(rand.NewSource(seed))
	}
	confirmedRequests := []modifiedClientRequest{}
	GLOBAL.allRequests.Lock()
	for uniqueID, possibleConfirmedRequest := range GLOBAL.allRequests.Buffered {
		allNeighborsConfirmed := true
		for neighbor := range GLOBAL.allRequests.ConfirmedIDs {
			_, ok := GLOBAL.allRequests.ConfirmedIDs[neighbor][uniqueID]
			if !ok {
				allNeighborsConfirmed = false
			}
		}
		if allNeighborsConfirmed {
			confirmedRequests = append(confirmedRequests, possibleConfirmedRequest)
			delete(GLOBAL.allRequests.Buffered, uniqueID)
			for neighbor := range GLOBAL.allRequests.ConfirmedIDs {
				delete(GLOBAL.allRequests.ConfirmedIDs[neighbor], uniqueID)
			}
		}
	}
	GLOBAL.allRequests.Unlock()
	GLOBAL.backendData.Unlock()

	sysHashes := make([]map[string]int64, GLOBAL.numberOfCores)
	for i := range sysHashes {
		sysHashes[i] = make(map[string]int64, totalNumSystems)
	}

	// TODO // verify that the order of objects is constant when deserialized
	parallelProcessSystems(futureGalaxy, func(ss *SolarSystem, coreID int) {

		// deserialize any systems that are in recovery
		val, exists := GLOBAL.backendData.SystemsInRecovery[ss.Name]
		if exists && val {
			proto.Unmarshal(GLOBAL.backendData.SystemBytes[ss.Name], ss)
		}
		rnum := rnumMapping[coreID]

		// calculate population
		for _, ob := range ss.Solarobjectslice {
			if isPlanet(ob.Type) || ob.Type == TITAN {
				ob.Population = growPopulation(rnum, timeMultiplier, ob.Population, ob.Food)
			}
		}

		// calculate a rough hash (must be done last)
		sysHashes[coreID][ss.Name] = ss.InterX + ss.InterY + int64(ss.Type)
		if len(ss.Solarobjectslice) != 0 {
			obj := ss.Solarobjectslice[rnum.Intn(len(ss.Solarobjectslice))]
			sysHashes[coreID][ss.Name] += obj.LocalX + obj.LocalY + int64(obj.Food)
			sysHashes[coreID][ss.Name] += int64(obj.Population) + int64(obj.Health)
		}

	})

	GLOBAL.backendData.Lock()
	// overwrite GLOBAL.backendData.SystemHashes
	for i := range sysHashes {
		for k, v := range sysHashes[i] {
			GLOBAL.backendData.SystemHashes[k] = v
		}
	}

	// reset systemsInRecovery now that we've processed them all
	GLOBAL.backendData.SystemsInRecovery = make(map[string]bool, totalNumSystems)
	GLOBAL.backendData.Unlock()

	logTelemetry(fName, fmt.Sprintf("finished in %v milliseconds",
		time.Since(start).Nanoseconds()/1e6))
}

func growPopulation(rnum *rand.Rand, timeMultiplier float64, currentIntPop uint64, foodSupply int32) uint64 {
	// https://play.golang.org/p/gWizLY0X2A
	N := float64(currentIntPop)
	foodSupplyDbl := float64(foodSupply)
	referencePopulation := float64(10000)
	maxPercentChange := .0005
	randomness := referencePopulation * (1 + rnum.NormFloat64()*.1)
	K := randomness * math.Pow(foodSupplyDbl, 3.4)

	logisticPercentChange := (K - N) / K
	switch {
	case logisticPercentChange < -1:
		logisticPercentChange = -1
	case logisticPercentChange > 1:
		logisticPercentChange = 1
	case logisticPercentChange == 0:
		logisticPercentChange = .000001
	}

	return uint64(N + ((N*maxPercentChange)*logisticPercentChange)*timeMultiplier)
}

func isPlanet(val int32) bool {
	if val < 12 {
		return true
	}
	return false
}

func isShip(val int32) bool {
	if (12 < val) && (val < 19) {
		return true
	}
	return false
}

// defining types of objects
const (
	TERRAN         int32 = iota // -20% population
	DESERT                      // -50% population, 2x logistics
	VOLCANIC                    // -75% population, lots of metal
	ICE                         // -50% population, lots of crystal
	BARREN                      // -90% population, 3x logistics
	FERROUS                     //
	GREENHOUSE                  //
	OCEANIC                     // max population
	LARGEASTEROID               //
	DWARF                       //
	ASTEROIDBELT                //
	PLASMASTORM                 //
	_                           // _12_
	LARGETRADESHIP              // spawns and moves automatically from trading nations
	FIGHTER                     // close range combat
	FLAKCORVETTE                // block point-defense damage
	COLONIZER                   // colonize a single planet in new system
	CAPITALSHIP                 // long range ship-to-ship combat
	TITAN                       // can colonize, block point-defense and engage long-range
	_                           // _19_
)

// solarsystem type
const (
	YELLOWGIANT int32 = iota // most desirable
	REDGIANT                 // mostly lava
	BLUEGIANT                // mostly ice
	NEUTRONSTAR              // just gas clouds
	WHITEDWARF               // just asteroids
	BLACKHOLE                // random, dangerous and AI spawn here
)

// galaxy type
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)
