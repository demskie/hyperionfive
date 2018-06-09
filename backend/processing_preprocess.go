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
		battle *gamestate.OngoingBattle
		siege  *gamestate.OngoingSiege
		action *gamestate.OutstandingAction
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
		galaxy.PointerMappings.fleetSlice[int(fleet.FleetID)] = fleet
	}
	for _, battle = range galaxy.Protobuf.Ongoingbattleslice {
		galaxy.PointerMappings.ongoingBattleMappings[battle.BattleID] = battle
		galaxy.PointerMappings.ongoingBattleSlice[int(battle.BattleID)] = battle
	}
	for _, siege = range galaxy.Protobuf.Ongoingsiegeslice {
		galaxy.PointerMappings.ongoingSiegeMappings[siege.SiegeID] = siege
		galaxy.PointerMappings.ongoingSiegeSlice[int(siege.SiegeID)] = siege
	}
	for _, action = range galaxy.Protobuf.Outstandingactionslice {
		galaxy.PointerMappings.outstandingActionMappings[action.ActionID] = action
		galaxy.PointerMappings.outstandingActionSlice[int(action.ActionID)] = action
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
			if planet.Owner == 0 {
				galaxy.NeutralAnyPlanetSlice = append(galaxy.NeutralAnyPlanetSlice, planet)
			}
			if planet.Owner == 0 && planet.ObjectType == gamestate.TERRAN {
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
			if planet.Owner == 0 {
				continue
			}
			_, exists = galaxy.PlayerFogOfWar[planet.Owner]
			if !exists {
				galaxy.PlayerFogOfWar[planet.Owner] = map[uint64]*gamestate.System{}
				galaxy.PlayerFogOfWar[planet.Owner][ss.NameUint] = ss
			} else if galaxy.PlayerFogOfWar[planet.Owner][ss.NameUint] != ss {
				galaxy.PlayerFogOfWar[planet.Owner][ss.NameUint] = ss
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
