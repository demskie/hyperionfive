package main

import (
	"math"
	"math/rand"
	"time"
)

func processGalaxy(seeds map[string]int64, requests []incomingClientRequest, galaxy Galaxy) {

	// TODO - verify that the order of objects is constant when deserialized

	for _, ss := range galaxy.Solarsystemslice {
		for _, ob := range ss.Solarobjectslice {
			if isPlanet(ob.Type) || ob.Type == TITAN {
				ob.Population = growPopulation(ob.Population, ob.Food)
			}
		}
	}
}

func growPopulation(currentIntPop uint64, foodSupply int32) uint64 {
	// https://play.golang.org/p/gWizLY0X2A

	N := float64(currentIntPop)
	//logDebug(commaFormatUint(uint64(N)))

	foodSupplyDbl := float64(foodSupply)
	stime := rand.NewSource(time.Now().UnixNano())
	rnum := rand.New(stime)

	// population if foodSupply == 0
	referencePopulation := float64(10000)

	// maximum % change allowed in single turn (.0005 is standard)
	maxPercentChange := .005

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

	return uint64(N + (N*maxPercentChange)*logisticPercentChange)
}

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////

// object type
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

////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////
