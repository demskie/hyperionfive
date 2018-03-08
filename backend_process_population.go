package main

import (
	"fmt"
	"math"
	"sync/atomic"
)

// CalculatePopulation ...
func (gal *GalaxyData) CalculatePopulation() {
	if gal.TickTime.HasBeenOneSecond == false {
		return
	}
	gal.ProcessingQueue <- func(coreNum int) {
		objectsPerCore := len(gal.PlanetPtrs) / GLOBAL.numberOfCores
		stopIndex := objectsPerCore * (coreNum + 1)
		startIndex := stopIndex - objectsPerCore
		if stopIndex > len(gal.PlanetPtrs) {
			stopIndex = len(gal.PlanetPtrs)
		}
		const (
			referencePopulation = float64(10000)
			maxPercentChange    = float64(.007)
		)
		var (
			N                     float64
			foodSupplyDbl         float64
			randomness            float64
			K                     float64
			logisticPercentChange float64
			fighterIncrement      int32
		)
		for objIndex := startIndex; objIndex < stopIndex; objIndex++ {
			// https://play.golang.org/p/gWizLY0X2A
			N = float64(gal.PlanetPtrs[objIndex].Population)
			if N < referencePopulation {
				N = referencePopulation
			}
			foodSupplyDbl = float64(1 + gal.PlanetPtrs[objIndex].Food)
			randomness = (referencePopulation + 123) * (1 + gal.RnumSlice[coreNum].NormFloat64()*.1)
			K = randomness * math.Pow(foodSupplyDbl, 3.4)
			switch gal.PlanetPtrs[objIndex].Type {
			case TERRAN:
				K *= 1
			case DESERT:
				K *= 0.9
			case VOLCANIC:
				K *= 0.8
			case ICE:
				K *= 0.8
			case BARREN:
				K *= 0.5
			case FERROUS:
				K *= 0.7
			case GREENHOUSE:
				K *= 0.7
			case OCEANIC:
				K *= 0.7
			case LARGEASTEROID:
				K *= 0.1
			case DWARF:
				K *= 0.2
			}
			logisticPercentChange = (K - N) / K
			switch {
			case logisticPercentChange < -1:
				logisticPercentChange = -1
			case logisticPercentChange > 1:
				logisticPercentChange = 1
			case logisticPercentChange == 0:
				logisticPercentChange = .000001 // can't divide by zero!
			}
			gal.PlanetPtrs[objIndex].Population = uint64(N + ((N * maxPercentChange) * logisticPercentChange))
			if gal.PlanetPtrs[objIndex].Population/100000 > uint64(gal.PlanetPtrs[objIndex].DockedFighterCount) {
				fighterIncrement = 1 + gal.RnumSlice[coreNum].Int31n(3)
				gal.PlanetPtrs[objIndex].DockedFighterCount += fighterIncrement
				backendCounters.IncrementFighterSpawnCount(fighterIncrement)
			}
			if objIndex%oneMillion == 0 {
				logDebug(fmt.Sprint(objIndex/oneMillion, gal.PlanetPtrs[objIndex].Food), commaFormatNum(gal.PlanetPtrs[objIndex].Population))
			}
			if false {
				logDebug(
					"\n\t\t\t", "timeMultiplier:", fmt.Sprint(gal.TimeMultiplier),
					"\n\t\t\t", "N:", commaFormatNum(uint64(N)),
					"\n\t\t\t", "foodSupplyDbl:", fmt.Sprint(foodSupplyDbl),
					"\n\t\t\t", "randomness:", fmt.Sprint(randomness),
					"\n\t\t\t", "K:", fmt.Sprint(K),
					"\n\t\t\t", "logisticPercentChange:", fmt.Sprint(logisticPercentChange),
					"\n\t\t\t", "after:", commaFormatNum(gal.PlanetPtrs[objIndex].Population),
					"\n\t\t\t", fmt.Sprintf("%v is more than %v",
						gal.PlanetPtrs[objIndex].Population/100000,
						uint64(gal.PlanetPtrs[objIndex].DockedFighterCount)))
			}
		}
	}
}

// IncreaseFoodOnNeutralPlanets ...
func (gal *GalaxyData) IncreaseFoodOnNeutralPlanets() {
	if gal.TickTime.HasBeenOneTenthSecond == false {
		return
	}
	numIterations := len(gal.NeutralPlanetPtrs) / GLOBAL.numberOfCores / 10 / 60 / 60
	gal.ProcessingQueue <- func(coreNum int) {
		var obj *SolarObject
		var foodCount int32
		var normalDistVal int32
		for i := 0; i < numIterations; i++ {
			obj = gal.NeutralPlanetPtrs[gal.RnumSlice[coreNum].Intn(len(gal.NeutralPlanetPtrs))]
			foodCount = atomic.LoadInt32(&obj.Food)
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			const stdDeviation = 21
			normalDistVal = int32(gal.RnumSlice[coreNum].NormFloat64() * stdDeviation)
			if absInt32(normalDistVal) >= foodCount {
				atomic.AddInt32(&obj.Food, 1)
			}
		}
	}
}
