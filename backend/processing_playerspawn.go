package backend

import (
	"fmt"

	"github.com/demskie/hyperionfive/backendmessage"
)

// SpawnNewPlayers ...
// TODO // ensure players have enough space to expand
// TODO // and spawn veteran players with more stuff
// BUG // frontend will need to resend playerspawn requests if they go unfulfilled
func (galaxy *GalaxyStruct) SpawnNewPlayers(universe *UniverseStruct) {
	const fName = "(spawnNewPlayers)"
	if galaxy.TickTime.HasBeenOneSecond == false ||
		len(universe.PendingPlayerSpawns) < 1 {
		return
	}
	log.Telemetry(fName, fmt.Sprintf("processing %v new player spawns", len(universe.PendingPlayerSpawns)))
	var (
		planet     *backendmessage.Planet
		playerData *playerInfo
		randIndex  int
	)
	for _, playerData = range universe.PendingPlayerSpawns {
		if len(galaxy.NeutralTerranPlanetSlice) < 1 {
			log.Error(fName, "ran out NeutralTerranPlanets")
			return
		}
		randIndex = galaxy.SingleRnum.Intn(len(galaxy.NeutralTerranPlanetSlice))
		planet = galaxy.NeutralTerranPlanetSlice[randIndex]
		planet.Food = int32(1 + playerData.strength)
		planet.Population = int64(10000 * playerData.strength * 10)
		//galaxy.Actors = append(galaxy.Actors, actor.New(playerData.nameUint))
		// BUG // not implemented
		// obj.ChangeOwner(playerData.uniqueID, galaxy)
	}
	universe.PendingPlayerSpawns = []*playerInfo{}
}
