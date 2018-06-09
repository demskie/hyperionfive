package backend

import (
	"fmt"
	"math"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
)

// SpawnFighters ...
//	func (galaxy *GalaxyStruct) SpawnFighters() {
//		const fName = "(spawnFighters)"
//		const imprecisionFactor = 10 * 10
//		if galaxy.TickTime.HasBeenOneTenthSecond == false {
//			return
//		}
//		galaxy.ProcessingQueue.Execute(func(coreNum int) {
//			startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
//				coreNum, args.NumberOfCores, len(galaxy.AnyPlanetSlice))
//			if stopIndex <= startIndex {
//				return
//			}
//			rnum := galaxy.ProcessingRnumSlice[coreNum]
//			startIndex += rnum.Intn(imprecisionFactor)
//			var (
//				obj               *gamestate.SolarObject
//				availableFighters int32
//			)
//			for i := startIndex; i < stopIndex; i += imprecisionFactor {
//				obj = galaxy.AnyPlanetSlice[i]
//				availableFighters = int32(obj.Population/100000) - obj.DockedFighterCount
//				if availableFighters > 0 {
//					availableFighters = rnum.Int31n(1 + (availableFighters / 13))
//					obj.DockedFighterCount += availableFighters
//					backendCounters.IncrementFighterSpawnCount(availableFighters)
//				}
//			}
//		})
//	}

// CalculatePopulation ...
func (galaxy *GalaxyStruct) CalculatePopulation() {
	const fName = "(calculatePopulation)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(galaxy.AnyPlanetSlice))
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
			planet                *gamestate.Planet
		)
		for planetIndex := startIndex; planetIndex < stopIndex; planetIndex += imprecisionFactor {
			planet = galaxy.AnyPlanetSlice[planetIndex]
			// https://play.golang.org/p/gWizLY0X2A
			N = float64(planet.Population)
			if N < referencePopulation {
				N = referencePopulation
			}
			foodSupplyDbl = float64(1 + planet.Food)
			randomness = (referencePopulation + 123) * (1 + rnum.NormFloat64()*.1)
			K = randomness * math.Pow(foodSupplyDbl, 3.4)
			switch planet.ObjectType {
			case gamestate.TERRAN:
				K *= 1
			case gamestate.DESERT:
				K *= 0.9
			case gamestate.VOLCANIC:
				K *= 0.8
			case gamestate.ICE:
				K *= 0.8
			case gamestate.BARREN:
				K *= 0.5
			case gamestate.FERROUS:
				K *= 0.7
			case gamestate.GREENHOUSE:
				K *= 0.7
			case gamestate.OCEANIC:
				K *= 0.7
			case gamestate.LARGEASTEROID:
				K *= 0.1
			case gamestate.DWARF:
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
			planet.Population = int64(N + ((N * maxPercentChange) * logisticPercentChange))
			if args.DebugModeBool {
				if planetIndex == 0 {
					log.Debug(planet.GetPlanetTypeString(), fmt.Sprint(planet.Food),
						intutil.CommaFormatNum(planet.Population))
				}
			}
		}
	})
}

// IncreaseFoodOnNeutralPlanets ...
func (galaxy *GalaxyStruct) IncreaseFoodOnNeutralPlanets() {
	const fName = "(increaseFoodOnNeutralPlanets)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(galaxy.NeutralAnyPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			planet        *gamestate.Planet
			normalDistVal int32
		)
		for i := startIndex; i < stopIndex; i += imprecisionFactor {
			planet = galaxy.NeutralAnyPlanetSlice[i]
			// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
			const stdDeviation = 21
			normalDistVal = int32(rnum.NormFloat64() * stdDeviation)
			if intutil.AbsInt32(normalDistVal) >= planet.Food {
				planet.Food++
			}
		}
	})
}
