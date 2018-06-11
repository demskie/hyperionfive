package backend

import (
	"fmt"

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
			maxPercentChange   = float64(.57)
			populationExponent = float64(1.8)
			randomFactor       = float64(.1)
		)
		var (
			N                     float64
			K                     float64
			logisticPercentChange float64
			planet                *gamestate.Planet
		)
		for planetIndex := startIndex; planetIndex < stopIndex; planetIndex += imprecisionFactor {
			planet = galaxy.AnyPlanetSlice[planetIndex]
			// https://play.golang.org/p/gWizLY0X2A
			if planet.Population == 0 {
				planet.Population = 100 + rnum.Int63n(2302)
			}
			N = float64(planet.Population)
			K = getCapacity(planet.PlanetType, planet.Food)
			K -= float64(rnum.Int63n(1 + int64(K*.05)))
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
						intutil.CommaFormatNum(planet.Population),
						intutil.CommaFormatNum(K))
				}
			}
		}
	})
}

func getCapacity(planetType, planetFood int32) (capacity float64) {
	switch planetType {
	case gamestate.TERRAN:
		capacity = 25 * oneBillion
	case gamestate.DESERT:
		capacity = 10 * oneBillion
	case gamestate.VOLCANIC:
		capacity = 3 * oneBillion
	case gamestate.ICE:
		capacity = 25 * oneBillion
	case gamestate.BARREN:
		capacity = 1 * oneBillion
	case gamestate.FERROUS:
		capacity = 5 * oneBillion
	case gamestate.GREENHOUSE:
		capacity = 20 * oneBillion
	case gamestate.OCEANIC:
		capacity = 39 * oneBillion
	case gamestate.LARGEASTEROID:
		capacity = oneBillion / 120
	case gamestate.DWARF:
		capacity = oneBillion / 5
	}
	switch planetFood {
	case 0:
		capacity /= 1000000
	case 1:
		capacity /= 100000
	case 2:
		capacity /= 40000
	case 3:
		capacity /= 20000
	case 4:
		capacity /= 13333
	case 5:
		capacity /= 10000
	case 6:
		capacity /= 4000
	case 7:
		capacity /= 2000
	case 8:
		capacity /= 1333
	case 9:
		capacity /= 1000
	case 10:
		capacity /= 800
	case 11:
		capacity /= 600
	case 12:
		capacity /= 400
	case 13:
		capacity /= 200
	case 14:
		capacity /= 100
	case 15:
		capacity /= 80
	case 16:
		capacity /= 70
	case 17:
		capacity /= 60
	case 18:
		capacity /= 50
	case 19:
		capacity /= 30
	case 20:
		capacity /= 14
	case 21:
		capacity /= 9
	case 22:
		capacity /= 5
	case 23:
		capacity /= 3
	case 24:
		capacity /= 2
	case 25:
		capacity /= 1.8
	case 26:
		capacity /= 1.7
	case 27:
		capacity /= 1.6
	case 28:
		capacity /= 1.5
	case 29:
		capacity /= 1.4
	case 30:
		capacity /= 1.3
	case 31:
		capacity /= 1.2
	case 32:
		capacity /= 1.1
	}
	return capacity
}

// IncreaseFoodOnNeutralPlanets ...
//	func (galaxy *GalaxyStruct) IncreaseFoodOnNeutralPlanets() {
//		const fName = "(increaseFoodOnNeutralPlanets)"
//		const imprecisionFactor = 10
//		if galaxy.TickTime.HasBeenOneSecond == false {
//			return
//		}
//		galaxy.ProcessingQueue.Execute(func(coreNum int) {
//			startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
//				coreNum, args.NumberOfCores, len(galaxy.NeutralAnyPlanetSlice))
//			if stopIndex <= startIndex {
//				return
//			}
//			rnum := galaxy.ProcessingRnumSlice[coreNum]
//			startIndex += rnum.Intn(imprecisionFactor)
//			var (
//				planet        *gamestate.Planet
//				normalDistVal int32
//			)
//			for i := startIndex; i < stopIndex; i += imprecisionFactor {
//				planet = galaxy.NeutralAnyPlanetSlice[i]
//				// https://homepage.divms.uiowa.edu/~mbognar/applets/normal.html
//				const stdDeviation = 21
//				normalDistVal = int32(rnum.NormFloat64() * stdDeviation)
//				if intutil.AbsInt32(normalDistVal) >= planet.Food {
//					planet.Food++
//				}
//			}
//		})
//	}
