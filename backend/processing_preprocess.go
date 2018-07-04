package backend

import "github.com/demskie/hyperionfive/gamestate"

// PreprocessPopulationObjects ...
func (galaxy *GalaxyStruct) PreprocessPopulationObjects() {
	const fName = "(preprocessPopulationObjects)"
	log.Debug(fName, "starting")
	var (
		ss     *gamestate.System
		planet *gamestate.Planet
	)
	for _, ss = range galaxy.Protobuf.Systemslice {
		for _, planet = range ss.Planetslice {
			galaxy.AnyPlanetSlice = append(galaxy.AnyPlanetSlice, planet)
			if planet.OwnerUint == 0 {
				galaxy.NeutralAnyPlanetSlice = append(galaxy.NeutralAnyPlanetSlice, planet)
			}
			if planet.OwnerUint == 0 && planet.PlanetType == gamestate.TERRAN {
				galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, planet)
			}
		}
	}
}

// PreprocessPlayerFogOfWar ...
func (galaxy *GalaxyStruct) PreprocessPlayerFogOfWar() {
	const fName = "(preprocessPlayerFogOfWar)"
	log.Debug(fName, "starting")
	var (
		ss     *gamestate.System
		planet *gamestate.Planet
		exists bool
	)
	for _, ss = range galaxy.Protobuf.Systemslice {
		for _, planet = range ss.Planetslice {
			if planet.OwnerUint == 0 {
				continue
			}
			_, exists = galaxy.PlayerFogOfWar[planet.OwnerUint]
			if !exists {
				galaxy.PlayerFogOfWar[planet.OwnerUint] = map[uint64]*gamestate.System{}
				galaxy.PlayerFogOfWar[planet.OwnerUint][ss.NameUint] = ss
			} else if galaxy.PlayerFogOfWar[planet.OwnerUint][ss.NameUint] != ss {
				galaxy.PlayerFogOfWar[planet.OwnerUint][ss.NameUint] = ss
			}
		}
	}
}
