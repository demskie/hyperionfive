package backend

import (
	"sync/atomic"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
)

// ProcessOngoingBattles ...
func (galaxy *GalaxyStruct) ProcessOngoingBattles() {
	const fName = "(processBattles)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	deleteBattle := make([][]*gamestate.OngoingBattle, args.NumberOfCores)
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(galaxy.Protobuf.Ongoingbattleslice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex = rnum.Intn(imprecisionFactor)

		var (
			battle                               *gamestate.OngoingBattle
			disadvantageScaler, lossesMultiplier float64
			aggressorStrength, defenderStrength  int64
			totalStrength, idx, failedIterations int64
			fleetID                              int32
			fleet                                *gamestate.Fleet
		)

		deleteBattle[coreNum] = []*gamestate.OngoingBattle{}

		for battleIndex := startIndex; battleIndex < stopIndex; battleIndex += imprecisionFactor {
			battle = galaxy.Protobuf.Ongoingbattleslice[battleIndex]
			if battle.Strategy == RUSHING {
				disadvantageScaler = float64(0.72)
				lossesMultiplier = float64(0.06)
			} else if battle.Strategy == ATTACKING {
				disadvantageScaler = float64(0.93)
				lossesMultiplier = float64(0.01)
			}
			aggressorStrength = 0
			for _, fleetID = range battle.AttackingFleetIDs {
				aggressorStrength += int64(atomic.LoadInt32(
					&galaxy.Protobuf.Fleetslice[fleetID].FighterCount))
				aggressorStrength += int64(atomic.LoadInt32(
					&galaxy.Protobuf.Fleetslice[fleetID].CapitalCount) * 100)
			}
			defenderStrength = 0
			for _, fleetID = range battle.DefendingFleetIDs {
				defenderStrength += int64(atomic.LoadInt32(
					&galaxy.Protobuf.Fleetslice[fleetID].FighterCount))
				defenderStrength += int64(atomic.LoadInt32(
					&galaxy.Protobuf.Fleetslice[fleetID].CapitalCount) * 100)
			}

			// BUG // with this implementation it's possible to have a negative number
			// of ships if a fleet is involved in more than one battle at a time

			// if true the Aggressor wins and the Defender loses forces
			aggressorStrength = int64(float64(aggressorStrength) * disadvantageScaler)
			totalStrength = aggressorStrength * defenderStrength
			if rnum.Int63n(totalStrength) < aggressorStrength {
				failedIterations = 0
				defenderStrength = 1 + int64(float64(defenderStrength)*lossesMultiplier)
				fleetID = int32(rnum.Intn(len(battle.DefendingFleetIDs)))
				for {
					if defenderStrength <= 0 {
						break
					}
					fleet = galaxy.GetFleetFromID(fleetID)
					if atomic.LoadInt32(&fleet.FighterCount) > 0 {
						atomic.AddInt32(&fleet.FighterCount, -1)
						defenderStrength--
						failedIterations = 0
					} else if atomic.LoadInt32(&fleet.CapitalCount) > 0 && defenderStrength >= 100 {
						atomic.AddInt32(&fleet.CapitalCount, -1)
						defenderStrength -= 100
						failedIterations = 0
					}
					idx++
					if idx >= int64(len(battle.DefendingFleetIDs)) {
						idx = 0
					}
					failedIterations++
					if failedIterations >= int64(len(battle.DefendingFleetIDs)) {
						deleteBattle[coreNum] = append(deleteBattle[coreNum], battle)
						break
					}
				}
			} else {
				failedIterations = 0
				aggressorStrength = 1 + int64(float64(aggressorStrength)*lossesMultiplier)
				fleetID = int32(rnum.Intn(len(battle.AttackingFleetIDs)))
				for {
					if aggressorStrength <= 0 {
						break
					}
					fleet = galaxy.GetFleetFromID(fleetID)
					if atomic.LoadInt32(&fleet.FighterCount) > 0 {
						atomic.AddInt32(&fleet.FighterCount, -1)
						aggressorStrength--
						failedIterations = 0
					} else if atomic.LoadInt32(&fleet.CapitalCount) > 0 && aggressorStrength >= 100 {
						atomic.AddInt32(&fleet.CapitalCount, -1)
						aggressorStrength -= 100
						failedIterations = 0
					}
					idx++
					if idx >= int64(len(battle.AttackingFleetIDs)) {
						idx = 0
					}
					failedIterations++
					if failedIterations >= int64(len(battle.AttackingFleetIDs)) {
						deleteBattle[coreNum] = append(deleteBattle[coreNum], battle)
						break
					}
				}
			}
		}
	})

	for i := range deleteBattle {
		for _, battle := range deleteBattle[i] {
			galaxy.DeleteOngoingBattle(battle)
		}
	}
}
