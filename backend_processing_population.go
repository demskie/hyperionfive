package main

import (
	"fmt"
	"math"
)

// SpawnFighters ...
func (galaxy *GalaxyStruct) SpawnFighters() {
	const fName = "(spawnFighters)"
	const imprecisionFactor = 10 * 10
	if galaxy.TickTime.HasBeenOneTenthSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(galaxy.AnyPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			obj               *SolarObject
			availableFighters int32
		)
		for i := startIndex; i < stopIndex; i += imprecisionFactor {
			obj = galaxy.AnyPlanetSlice[i]
			availableFighters = int32(obj.Population/100000) - obj.DockedFighterCount
			if availableFighters > 0 {
				availableFighters = rnum.Int31n(1 + (availableFighters / 13))
				obj.DockedFighterCount += availableFighters
				backendCounters.IncrementFighterSpawnCount(availableFighters)
			}
		}
	})
}

// CalculatePopulation ...
func (galaxy *GalaxyStruct) CalculatePopulation() {
	const fName = "(calculatePopulation)"
	const imprecisionFactor = 10 * 1
	if galaxy.TickTime.HasBeenOneTenthSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(galaxy.AnyPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)

		const (
			referencePopulation = float64(10000)
			maxPercentChange    = float64(.07)
		)
		var (
			N                     float64
			foodSupplyDbl         float64
			randomness            float64
			K                     float64
			logisticPercentChange float64
			obj                   *SolarObject
		)
		for objIndex := startIndex; objIndex < stopIndex; objIndex += imprecisionFactor {
			obj = galaxy.AnyPlanetSlice[objIndex]
			// https://play.golang.org/p/gWizLY0X2A
			N = float64(obj.Population)
			if N < referencePopulation {
				N = referencePopulation
			}
			foodSupplyDbl = float64(1 + obj.Food)
			randomness = (referencePopulation + 123) * (1 + rnum.NormFloat64()*.1)
			K = randomness * math.Pow(foodSupplyDbl, 3.4)
			switch obj.Type {
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
			obj.Population = uint64(N + ((N * maxPercentChange) * logisticPercentChange))
			if GLOBAL.debugModeBool {
				if objIndex == 0 {
					logDebug(fmt.Sprint(objIndex/oneMillion, obj.Food), commaFormatNum(obj.Population))
				}
			}
			if false {
				logDebug(
					"\n\t\t\t", "timeMultiplier:", fmt.Sprint(galaxy.TimeMultiplier),
					"\n\t\t\t", "N:", commaFormatNum(uint64(N)),
					"\n\t\t\t", "foodSupplyDbl:", fmt.Sprint(foodSupplyDbl),
					"\n\t\t\t", "randomness:", fmt.Sprint(randomness),
					"\n\t\t\t", "K:", fmt.Sprint(K),
					"\n\t\t\t", "logisticPercentChange:", fmt.Sprint(logisticPercentChange),
					"\n\t\t\t", "after:", commaFormatNum(obj.Population),
					"\n\t\t\t", fmt.Sprintf("%v is more than %v", obj.Population/100000, uint64(obj.DockedFighterCount)))
			}
		}
	})
}

// IncreaseFoodOnNeutralPlanets ...
func (galaxy *GalaxyStruct) IncreaseFoodOnNeutralPlanets() {
	const fName = "(increaseFoodOnNeutralPlanets)"
	const imprecisionFactor = 10 * 30
	if galaxy.TickTime.HasBeenOneTenthSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := getStartAndStopIndexForCore(coreNum, len(galaxy.NeutralAnyPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			obj           *SolarObject
			normalDistVal int32
		)
		for i := startIndex; i < stopIndex; i += imprecisionFactor {
			obj = galaxy.NeutralAnyPlanetSlice[i]
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			const stdDeviation = 21
			normalDistVal = int32(rnum.NormFloat64() * stdDeviation)
			if absInt32(normalDistVal) >= obj.Food {
				obj.Food++
			}
		}
	})
}
