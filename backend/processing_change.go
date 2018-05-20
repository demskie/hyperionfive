package backend

import (
	"fmt"

	"github.com/demskie/hyperionfive/gamestate"
)

// ChangeOwner ...
func (galaxy *GalaxyStruct) ChangeOwner(obj *gamestate.SolarObject, newOwner int32) {
	galaxy.PlayerFogOfWarMutex.Lock()
	defer galaxy.PlayerFogOfWarMutex.Unlock()
	if obj.Owner == newOwner {
		log.Error(fmt.Sprintf("obj.ChangeOwner called unnecessarily on '%v'", obj.Owner))
		return
	}
	oldOwner := obj.Owner
	obj.Owner = newOwner

	if oldOwner != 0 {
		_, playerExists := galaxy.PlayerFogOfWar[oldOwner]
		if playerExists {
			removeEntireSystem := true
			systemInQuestion := galaxy.GetSolarSystemFromName(obj.ParentSystemNameUint)
			for _, solarObject := range systemInQuestion.Solarobjectslice {
				if solarObject.Owner == oldOwner {
					removeEntireSystem = false
					break
				}
			}
			if removeEntireSystem {
				delete(galaxy.PlayerFogOfWar[oldOwner], obj.ParentSystemNameUint)
			}
			if len(galaxy.PlayerFogOfWar[oldOwner]) == 0 {
				delete(galaxy.PlayerFogOfWar, oldOwner)
			}
		}
	}

	if oldOwner == 0 && obj.IsPlanet() {
		galaxy.NeutralTerranPlanetMutex.Lock()
		slc := galaxy.NeutralTerranPlanetSlice
		for idx, anObj := range galaxy.NeutralTerranPlanetSlice {
			if anObj == obj {
				slc[idx] = slc[len(slc)-1]
				slc[len(slc)-1] = nil
				galaxy.NeutralTerranPlanetSlice = slc[:len(slc)-1]
				break
			}
		}
		galaxy.NeutralTerranPlanetMutex.Unlock()
	}

	if newOwner != 0 {
		ss := galaxy.GetSolarSystemFromName(obj.ParentSystemNameUint)
		_, playerExists := galaxy.PlayerFogOfWar[newOwner]
		if playerExists {
			galaxy.PlayerFogOfWar[newOwner][obj.ParentSystemNameUint] = ss
		} else {
			galaxy.PlayerFogOfWar[newOwner] = map[uint64]*gamestate.SolarSystem{obj.ParentSystemNameUint: ss}
		}
	}

	if newOwner == 0 && obj.IsPlanet() {
		galaxy.NeutralTerranPlanetMutex.Lock()
		galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, obj)
		galaxy.NeutralTerranPlanetMutex.Unlock()
	}

	return
}
