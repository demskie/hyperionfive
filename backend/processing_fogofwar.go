package backend

import "github.com/demskie/hyperionfive/gamestate"

// PlayerFogOfWarSlice ...
func (galaxy *GalaxyStruct) PlayerFogOfWarSlice(playerName uint64) []*gamestate.System {
	systemSlice := make([]*gamestate.System, 0, len(galaxy.PlayerFogOfWar[playerName]))
	for _, system := range galaxy.PlayerFogOfWar[playerName] {
		systemSlice = append(systemSlice, system)
	}
	return systemSlice
}
