package backend

import (
	"fmt"

	"github.com/demskie/hyperionfive/backendmessage"
)

// ChangePlanetOwner ...
func (galaxy *GalaxyStruct) ChangePlanetOwner(planet *backendmessage.Planet, newOwnerUint uint64) {
	// galaxy.PlayerFogOfWarMutex.Lock()
	// defer galaxy.PlayerFogOfWarMutex.Unlock()
	if planet.OwnerUint == newOwnerUint {
		log.Error(fmt.Sprintf("galaxy.ChangePlanetOwner called unnecessarily on '%v'", planet.OwnerUint))
		return
	}
	oldOwnerUint := planet.OwnerUint
	planet.OwnerUint = newOwnerUint

	//	if oldOwnerUint != 0 {
	//		_, playerExists := galaxy.PlayerFogOfWar[oldOwnerUint]
	//		if playerExists {
	//			removeEntireSystem := true
	//			systemInQuestion := galaxy.LookupSystemName(planet.ParentSystemUint)
	//			for _, pl := range systemInQuestion.Planetslice {
	//				if pl.OwnerUint == oldOwnerUint {
	//					removeEntireSystem = false
	//					break
	//				}
	//			}
	//			if removeEntireSystem {
	//				delete(galaxy.PlayerFogOfWar[oldOwnerUint], planet.ParentSystemUint)
	//			}
	//			if len(galaxy.PlayerFogOfWar[oldOwnerUint]) == 0 {
	//				delete(galaxy.PlayerFogOfWar, oldOwnerUint)
	//			}
	//		}
	//	}

	if oldOwnerUint == 0 {
		slc := galaxy.NeutralTerranPlanetSlice
		for idx, aPlanet := range galaxy.NeutralTerranPlanetSlice {
			if aPlanet == planet {
				slc[idx] = slc[len(slc)-1]
				slc[len(slc)-1] = nil
				galaxy.NeutralTerranPlanetSlice = slc[:len(slc)-1]
				break
			}
		}
	}

	if newOwnerUint != 0 {
		//	ss := galaxy.LookupSystemName(planet.ParentSystemUint)
		//	_, playerExists := galaxy.PlayerFogOfWar[newOwnerUint]
		//	if playerExists {
		//		galaxy.PlayerFogOfWar[newOwnerUint][planet.ParentSystemUint] = ss
		//	} else {
		//		galaxy.PlayerFogOfWar[newOwnerUint] = map[uint64]*backendmessage.System{planet.ParentSystemUint: ss}
		//	}
	} else {
		galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, planet)
	}

	return
}
