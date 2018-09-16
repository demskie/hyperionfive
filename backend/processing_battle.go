package backend

import (
	"github.com/demskie/hyperionfive/gamestate"
)

// ProcessOngoingBattles ...
func (galaxy *GalaxyStruct) ProcessOngoingBattles() {
	const fName = "(processBattles)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	var (
		disadvantageScaler, lossesMultiplier float64
		aggressorStrength, defenderStrength  int64
		totalStrength                        int64
		fleetID                              int32
		fleet                                *gamestate.Fleet
	)
	rnum := galaxy.ProcessingRnumSlice[0]
	for _, battle := range galaxy.Protobuf.Battle {
		if battle.Strategy == RUSHING {
			disadvantageScaler = float64(0.72)
			lossesMultiplier = float64(0.06)
		} else if battle.Strategy == ATTACKING {
			disadvantageScaler = float64(0.93)
			lossesMultiplier = float64(0.01)
		}
		aggressorStrength = 0
		for fleetID = range battle.AttackingFleetIDs {
			aggressorStrength += int64(galaxy.Protobuf.Fleet[fleetID].FighterCount)
			aggressorStrength += int64(galaxy.Protobuf.Fleet[fleetID].CapitalCount * 100)
		}
		defenderStrength = 0
		for fleetID = range battle.DefendingFleetIDs {
			defenderStrength += int64(galaxy.Protobuf.Fleet[fleetID].FighterCount)
			defenderStrength += int64(galaxy.Protobuf.Fleet[fleetID].CapitalCount * 100)
		}
		aggressorStrength = int64(float64(aggressorStrength) * disadvantageScaler)
		totalStrength = aggressorStrength * defenderStrength
		if rnum.Int63n(totalStrength) < aggressorStrength {
			defenderStrength = 1 + int64(float64(defenderStrength)*lossesMultiplier)
			fleetID = int32(rnum.Intn(len(battle.DefendingFleetIDs)))
			fleet = galaxy.Protobuf.Fleet[fleetID]
			for defenderStrength > 0 {
				if fleet.FighterCount > 0 {
					fleet.FighterCount--
					defenderStrength--
				} else if fleet.CapitalCount > 0 && defenderStrength >= 100 {
					fleet.CapitalCount--
					defenderStrength -= 100
				}
			}
		} else {
			aggressorStrength = 1 + int64(float64(aggressorStrength)*lossesMultiplier)
			fleetID = int32(rnum.Intn(len(battle.AttackingFleetIDs)))
			fleet = galaxy.Protobuf.Fleet[fleetID]
			for aggressorStrength > 0 {
				if fleet.FighterCount > 0 {
					fleet.FighterCount--
					aggressorStrength--
				} else if fleet.CapitalCount > 0 && aggressorStrength >= 100 {
					fleet.CapitalCount--
					aggressorStrength -= 100
				}
			}
		}
	}
}
