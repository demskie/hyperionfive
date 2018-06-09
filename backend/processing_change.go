package backend

import (
	"fmt"

	"github.com/demskie/hyperionfive/gamestate"
)

// ChangePlanetOwner ...
func (galaxy *GalaxyStruct) ChangePlanetOwner(planet *gamestate.Planet, newOwner int32) {
	galaxy.PlayerFogOfWarMutex.Lock()
	defer galaxy.PlayerFogOfWarMutex.Unlock()
	if planet.Owner == newOwner {
		log.Error(fmt.Sprintf("galaxy.ChangePlanetOwner called unnecessarily on '%v'", planet.Owner))
		return
	}
	oldOwner := planet.Owner
	planet.Owner = newOwner

	if oldOwner != 0 {
		_, playerExists := galaxy.PlayerFogOfWar[oldOwner]
		if playerExists {
			removeEntireSystem := true
			systemInQuestion := galaxy.GetSystemFromName(planet.ParentSystemNameUint)
			for _, pl := range systemInQuestion.Planetslice {
				if pl.Owner == oldOwner {
					removeEntireSystem = false
					break
				}
			}
			if removeEntireSystem {
				delete(galaxy.PlayerFogOfWar[oldOwner], planet.ParentSystemNameUint)
			}
			if len(galaxy.PlayerFogOfWar[oldOwner]) == 0 {
				delete(galaxy.PlayerFogOfWar, oldOwner)
			}
		}
	}

	if oldOwner == 0 {
		galaxy.NeutralTerranPlanetMutex.Lock()
		slc := galaxy.NeutralTerranPlanetSlice
		for idx, aPlanet := range galaxy.NeutralTerranPlanetSlice {
			if aPlanet == planet {
				slc[idx] = slc[len(slc)-1]
				slc[len(slc)-1] = nil
				galaxy.NeutralTerranPlanetSlice = slc[:len(slc)-1]
				break
			}
		}
		galaxy.NeutralTerranPlanetMutex.Unlock()
	}

	if newOwner != 0 {
		ss := galaxy.GetSystemFromName(planet.ParentSystemNameUint)
		_, playerExists := galaxy.PlayerFogOfWar[newOwner]
		if playerExists {
			galaxy.PlayerFogOfWar[newOwner][planet.ParentSystemNameUint] = ss
		} else {
			galaxy.PlayerFogOfWar[newOwner] = map[uint64]*gamestate.System{planet.ParentSystemNameUint: ss}
		}
	} else {
		galaxy.NeutralTerranPlanetMutex.Lock()
		galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, planet)
		galaxy.NeutralTerranPlanetMutex.Unlock()
	}

	return
}
