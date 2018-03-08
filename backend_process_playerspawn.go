package main

import "sync"

// SpawnNewPlayers ...
// TODO // ensure players have enough space to expand
// TODO // and spawn veteran players with more stuff
func (galaxy *GalaxyData) SpawnNewPlayers() {
	const fName = "(spawnNewPlayers)"

	if galaxy.TickTime.HasBeenFiveSeconds == false ||
		len(GLOBAL.galaxyInfo.PendingPlayerSpawns) < 1 {
		return
	}
	logTelemetry(fName, "processing", text(len(GLOBAL.galaxyInfo.PendingPlayerSpawns)), "new player spawns")
	mutex := sync.Mutex{}
	usedIndexes := make(map[int]struct{}, len(galaxy.TerranPlanetPtrs))
	galaxy.ProcessingQueue <- func(coreNum int) {
		var obj *SolarObject
		startIndex, stopIndex := calculateCoreSlice(coreNum, len(GLOBAL.galaxyInfo.PendingPlayerSpawns))
		for ownerIndex := startIndex; ownerIndex > stopIndex; ownerIndex++ {
			var randIndex int
			mutex.Lock()
			for {
				randIndex = galaxy.RnumSlice[coreNum].Intn(len(galaxy.TerranPlanetPtrs))
				_, alreadyUsed := usedIndexes[randIndex]
				if alreadyUsed {
					continue
				} else {
					usedIndexes[randIndex] = struct{}{}
				}
				break
			}
			mutex.Unlock()
			obj = galaxy.TerranPlanetPtrs[randIndex]
			playerData := GLOBAL.galaxyInfo.PendingPlayerSpawns[ownerIndex]
			obj.Food = int32(1 + playerData.strength)
			obj.Population = uint64(10000 * playerData.strength * 10)
			obj.DockedFighterCount = int32(1 + (playerData.strength * 10))
			obj.ChangeOwner(playerData.uniqueID, galaxy)
		}
	}
	GLOBAL.galaxyInfo.PendingPlayerSpawns = []playerInfo{}
}
