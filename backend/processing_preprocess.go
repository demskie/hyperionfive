package backend

import "github.com/demskie/hyperionfive/gamestate"

// PreprocessPointerMappings ...
func (galaxy *GalaxyStruct) PreprocessPointerMappings() {
	const fName = "(preprocessPointerMappings)"
	log.Debug(fName, "starting")
	var (
		ss     *gamestate.System
		planet *gamestate.Planet
		object *gamestate.Object
		fleet  *gamestate.Fleet
		battle *gamestate.Battle
		siege  *gamestate.Siege
		action *gamestate.Action
	)
	for _, ss = range galaxy.Protobuf.Systemslice {
		for _, planet = range ss.Planetslice {
			galaxy.PointerMappings.planetMappings[planet.NameUint] = planet
		}
		for _, object = range ss.Objectslice {
			galaxy.PointerMappings.objectMappings[object.NameUint] = object
		}
		galaxy.PointerMappings.systemMappings[ss.NameUint] = ss
	}
	for _, fleet = range galaxy.Protobuf.Fleetslice {
		galaxy.PointerMappings.fleetMappings[fleet.FleetID] = fleet
	}
	for _, battle = range galaxy.Protobuf.Battleslice {
		galaxy.PointerMappings.battleMappings[battle.BattleID] = battle
	}
	for _, siege = range galaxy.Protobuf.Siegeslice {
		galaxy.PointerMappings.siegeMappings[siege.SiegeID] = siege
	}
	for _, action = range galaxy.Protobuf.Actionslice {
		galaxy.PointerMappings.actionMappings[action.ActionID] = action
	}
}

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
			if planet.OwnerUint == 0 && planet.ObjectType == gamestate.TERRAN {
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

// PreprocessAllocations ...
func (galaxy *GalaxyStruct) PreprocessAllocations() {
	const fName = "(preprocessAllocations)"
	log.Debug(fName, "starting")
	for _, ss := range galaxy.Protobuf.Systemslice {
		galaxy.SystemNameLookup[ss.NameUint] = ss
	}
}
