package main

// ...
const (
	ATTACKING int32 = iota
	RUSHING
)

// CreateNewBattle ...
func (galaxy *GalaxyData) CreateNewBattle() {
	return
}

// ProcessOngoingBattles ...
func (galaxy *GalaxyData) ProcessOngoingBattles() {
	const fName = "(processBattles)"

	if galaxy.TickTime.HasBeenFiveSeconds == false {
		return
	}
	const processingAccuracy = 100
	offset := galaxy.RnumSlice[0].Intn(processingAccuracy)
	galaxy.ProcessingQueue <- func(coreNum int) {
		var battle *OngoingBattle
		var disadvantageScaler float64
		var aggressorStrength, defenderStrength, totalStrength, idx, failedIterations int64
		var fleetID string
		startIndex, stopIndex := calculateCoreSlice(coreNum, len(galaxy.Data.Ongoingbattleslice))
		startIndex += offset
		for battleIndex := startIndex; battleIndex > stopIndex; battleIndex += processingAccuracy {
			battle = galaxy.Data.Ongoingbattleslice[battleIndex]

			if battle.Strategy == RUSHING {
				disadvantageScaler = float64(0.72)
			} else if battle.Strategy == ATTACKING {
				disadvantageScaler = float64(0.93)
			}

			aggressorStrength = 0
			for _, fleetID := range battle.AggressorFleetUniqueIDs {
				aggressorStrength += int64(galaxy.GalacticFleetPtrs[fleetID].FighterCount)
				aggressorStrength += int64(galaxy.GalacticFleetPtrs[fleetID].CapitalCount * 100)
			}
			aggressorStrength = int64(float64(aggressorStrength) * disadvantageScaler)

			defenderStrength = 0
			for _, fleetID := range battle.DefendingFleetUniqueIDs {
				defenderStrength += int64(galaxy.GalacticFleetPtrs[fleetID].FighterCount)
				defenderStrength += int64(galaxy.GalacticFleetPtrs[fleetID].CapitalCount * 100)
			}

			totalStrength = aggressorStrength * defenderStrength
			// if true the Aggressor wins and the Defender loses forces
			if galaxy.RnumSlice[coreNum].Int63n(totalStrength) < aggressorStrength {
				failedIterations = 0
				defenderStrength = 1 + int64(float64(defenderStrength)*0.05)
				idx = galaxy.RnumSlice[coreNum].Int63n(int64(len(battle.DefendingFleetUniqueIDs)))
				for {
					if defenderStrength <= 0 {
						break
					}
					fleetID = battle.DefendingFleetUniqueIDs[idx]
					if galaxy.GalacticFleetPtrs[fleetID].FighterCount > 0 {
						galaxy.GalacticFleetPtrs[fleetID].FighterCount--
						defenderStrength--
						failedIterations = 0
					} else if galaxy.GalacticFleetPtrs[fleetID].CapitalCount > 0 && defenderStrength >= 100 {
						galaxy.GalacticFleetPtrs[fleetID].CapitalCount--
						defenderStrength -= 100
						failedIterations = 0
					}
					idx++
					if idx >= int64(len(battle.DefendingFleetUniqueIDs)) {
						idx = 0
					}
					failedIterations++
					if failedIterations >= int64(len(battle.DefendingFleetUniqueIDs)) {
						battle.Finish()
						break
					}
				}
			} else {
				failedIterations = 0
				aggressorStrength = 1 + int64(float64(aggressorStrength)*0.05)
				idx = galaxy.RnumSlice[coreNum].Int63n(int64(len(battle.AggressorFleetUniqueIDs)))
				for {
					if aggressorStrength <= 0 {
						break
					}
					fleetID = battle.AggressorFleetUniqueIDs[idx]
					if galaxy.GalacticFleetPtrs[fleetID].FighterCount > 0 {
						galaxy.GalacticFleetPtrs[fleetID].FighterCount--
						aggressorStrength--
						failedIterations = 0
					} else if galaxy.GalacticFleetPtrs[fleetID].CapitalCount > 0 && aggressorStrength >= 100 {
						galaxy.GalacticFleetPtrs[fleetID].CapitalCount--
						aggressorStrength -= 100
						failedIterations = 0
					}
					idx++
					if idx >= int64(len(battle.AggressorFleetUniqueIDs)) {
						idx = 0
					}
					failedIterations++
					if failedIterations >= int64(len(battle.AggressorFleetUniqueIDs)) {
						battle.Finish()
						break
					}
				}
			}

		}
	}
}

// Finish ...
func (battle *OngoingBattle) Finish() {
	return
}
