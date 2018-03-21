package main

import "fmt"

// ChangeOwner ...
func (obj *SolarObject) ChangeOwner(newOwner string, galaxy *GalaxyStruct) {
	galaxy.PlayerFogOfWarMutex.Lock()
	defer galaxy.PlayerFogOfWarMutex.Unlock()
	if obj.Owner == newOwner {
		logError(fmt.Sprintf("obj.ChangeOwner called unnecessarily on '%s'", obj.Owner))
		return
	}
	oldOwner := obj.Owner
	obj.Owner = newOwner

	if oldOwner != "" {
		_, playerExists := galaxy.PlayerFogOfWar[oldOwner]
		if playerExists {
			removeEntireSystem := true
			systemInQuestion := galaxy.GetSolarSystemFromName(obj.ParentSystemName)
			for _, solarObject := range systemInQuestion.Solarobjectslice {
				if solarObject.Owner == oldOwner {
					removeEntireSystem = false
					break
				}
			}
			if removeEntireSystem {
				delete(galaxy.PlayerFogOfWar[oldOwner], obj.ParentSystemName)
			}
			if len(galaxy.PlayerFogOfWar[oldOwner]) == 0 {
				delete(galaxy.PlayerFogOfWar, oldOwner)
			}
		}
	}

	if oldOwner == "" && obj.IsPlanet() {
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

	if newOwner != "" {
		ss := galaxy.GetSolarSystemFromName(obj.ParentSystemName)
		_, playerExists := galaxy.PlayerFogOfWar[newOwner]
		if playerExists {
			galaxy.PlayerFogOfWar[newOwner][obj.ParentSystemName] = ss
		} else {
			galaxy.PlayerFogOfWar[newOwner] = map[string]*SolarSystem{obj.ParentSystemName: ss}
		}
	}

	if newOwner == "" && obj.IsPlanet() {
		galaxy.NeutralTerranPlanetMutex.Lock()
		galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, obj)
		galaxy.NeutralTerranPlanetMutex.Unlock()
	}

	return
}
