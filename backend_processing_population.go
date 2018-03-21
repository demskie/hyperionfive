package main

import (
	"fmt"
	"math"
	"sync/atomic"
)

// CalculatePopulation ...
func (galaxy *GalaxyStruct) CalculatePopulation() {
	const fName = "(calculatePopulation)"
	const imprecisionFactor = 1000
	if galaxy.TickTime.HasBeenOneHundrethSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(galaxy.PopulationObjectsSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex = rnum.Intn(imprecisionFactor)

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
		for objIndex := startIndex; objIndex < stopIndex; objIndex += imprecisionFactor {
			// https://play.golang.org/p/gWizLY0X2A
			N = float64(galaxy.PopulationObjectsSlice[objIndex].Population)
			if N < referencePopulation {
				N = referencePopulation
			}
			foodSupplyDbl = float64(1 + galaxy.PopulationObjectsSlice[objIndex].Food)
			randomness = (referencePopulation + 123) * (1 + galaxy.ProcessingRnumSlice[coreNum].NormFloat64()*.1)
			K = randomness * math.Pow(foodSupplyDbl, 3.4)
			switch galaxy.PopulationObjectsSlice[objIndex].Type {
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
			galaxy.PopulationObjectsSlice[objIndex].Population = uint64(N + ((N * maxPercentChange) * logisticPercentChange))
			if galaxy.PopulationObjectsSlice[objIndex].Population/100000 > uint64(galaxy.PopulationObjectsSlice[objIndex].DockedFighterCount) {
				fighterIncrement = 1 + galaxy.ProcessingRnumSlice[coreNum].Int31n(3)
				galaxy.PopulationObjectsSlice[objIndex].DockedFighterCount += fighterIncrement
				backendCounters.IncrementFighterSpawnCount(fighterIncrement)
			}
			if objIndex%oneMillion == 0 {
				logDebug(fmt.Sprint(objIndex/oneMillion, galaxy.PopulationObjectsSlice[objIndex].Food), commaFormatNum(galaxy.PopulationObjectsSlice[objIndex].Population))
			}
			if false {
				logDebug(
					"\n\t\t\t", "timeMultiplier:", fmt.Sprint(galaxy.TimeMultiplier),
					"\n\t\t\t", "N:", commaFormatNum(uint64(N)),
					"\n\t\t\t", "foodSupplyDbl:", fmt.Sprint(foodSupplyDbl),
					"\n\t\t\t", "randomness:", fmt.Sprint(randomness),
					"\n\t\t\t", "K:", fmt.Sprint(K),
					"\n\t\t\t", "logisticPercentChange:", fmt.Sprint(logisticPercentChange),
					"\n\t\t\t", "after:", commaFormatNum(galaxy.PopulationObjectsSlice[objIndex].Population),
					"\n\t\t\t", fmt.Sprintf("%v is more than %v",
						galaxy.PopulationObjectsSlice[objIndex].Population/100000,
						uint64(galaxy.PopulationObjectsSlice[objIndex].DockedFighterCount)))
			}
		}
	})
}

// IncreaseFoodOnNeutralPlanets ...
func (galaxy *GalaxyStruct) IncreaseFoodOnNeutralPlanets() {
	const fName = "(increaseFoodOnNeutralPlanets)"
	const imprecisionFactor = 1000
	if galaxy.TickTime.HasBeenOneHundrethSecond == false {
		return
	}
	galaxy.PopulationObjectsMutex.Lock()
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(galaxy.NeutralTerranPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex = rnum.Intn(imprecisionFactor)

		var (
			obj                           *SolarObject
			foodCount, normalDistVal, idx int32
		)
		for i := startIndex; i < stopIndex; i += imprecisionFactor {
			idx = galaxy.ProcessingRnumSlice[coreNum].Int31n(int32(len(galaxy.NeutralTerranPlanetSlice)))
			obj = galaxy.NeutralTerranPlanetSlice[idx]
			foodCount = atomic.LoadInt32(&obj.Food)
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			const stdDeviation = 21
			normalDistVal = int32(galaxy.ProcessingRnumSlice[coreNum].NormFloat64() * stdDeviation)
			if absInt32(normalDistVal) >= foodCount {
				atomic.AddInt32(&obj.Food, 1)
			}
		}
	})
	galaxy.PopulationObjectsMutex.Unlock()
}
