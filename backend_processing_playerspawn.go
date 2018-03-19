package main

// SpawnNewPlayers ...
// TODO // ensure players have enough space to expand
// TODO // and spawn veteran players with more stuff
// BUG // frontend will need to resend playerspawn requests if they go unfulfilled
func (galaxy *GalaxyStruct) SpawnNewPlayers() {
	const fName = "(spawnNewPlayers)"
	if galaxy.TickTime.HasBeenOneHalfSecond == false ||
		len(GLOBAL.galaxyInfo.PendingPlayerSpawns) < 1 {
		return
	}
	logTelemetry(fName, "processing", text(len(GLOBAL.galaxyInfo.PendingPlayerSpawns)), "new player spawns")
	var (
		obj        *SolarObject
		playerData *playerInfo
		randIndex  int
	)
	for _, playerData = range GLOBAL.galaxyInfo.PendingPlayerSpawns {
		if len(galaxy.NeutralTerranPlanetSlice) < 1 {
			logError(fName, "ran out TerranPlanetPtrs")
			return
		}
		galaxy.SingleRnumMutex.Lock()
		randIndex = galaxy.SingleRnum.Intn(len(galaxy.NeutralTerranPlanetSlice))
		galaxy.SingleRnumMutex.Unlock()
		obj = galaxy.NeutralTerranPlanetSlice[randIndex]
		obj.Food = int32(1 + playerData.strength)
		obj.Population = uint64(10000 * playerData.strength * 10)
		obj.DockedFighterCount = int32(1 + (playerData.strength * 10))
		// BUG // ChangeOwner must delete obj from NeutralTerranPlanetSlice
		obj.ChangeOwner(playerData.uniqueID, galaxy)
	}
	GLOBAL.galaxyInfo.PendingPlayerSpawns = []*playerInfo{}
	logDebug(fName, "finished")
	galaxy.NeutralTerranPlanetMutex.Unlock()
}
