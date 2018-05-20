package backend

import "github.com/demskie/hyperionfive/gamestate"

// PreprocessPointerMappings ...
func (galaxy *GalaxyStruct) PreprocessPointerMappings() {
	const fName = "(preprocessPointerMappings)"
	log.Debug(fName, "starting")
	var (
		ss     *gamestate.SolarSystem
		obj    *gamestate.SolarObject
		fleet  *gamestate.GalacticFleet
		battle *gamestate.OngoingBattle
		siege  *gamestate.OngoingSiege
		action *gamestate.OutstandingAction
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			galaxy.PointerMappings.solarObjectMappings[obj.NameUint] = obj
		}
		galaxy.PointerMappings.solarSystemMappings[ss.NameUint] = ss
	}
	for _, fleet = range galaxy.Protobuf.Galacticfleetslice {
		galaxy.PointerMappings.galacticFleetMappings[fleet.FleetID] = fleet
		galaxy.PointerMappings.galacticFleetSlice[int(fleet.FleetID)] = fleet
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
		ss  *gamestate.SolarSystem
		obj *gamestate.SolarObject
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.IsPlanet() {
				galaxy.AnyPlanetSlice = append(galaxy.AnyPlanetSlice, obj)
				if obj.Owner == 0 {
					galaxy.NeutralAnyPlanetSlice = append(galaxy.NeutralAnyPlanetSlice, obj)
				}
				if obj.Owner == 0 && obj.ObjectType == gamestate.TERRAN {
					galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, obj)
				}
			}
		}
	}
}

// PreprocessPlayerFogOfWar ...
func (galaxy *GalaxyStruct) PreprocessPlayerFogOfWar() {
	const fName = "(preprocessPlayerFogOfWar)"
	log.Debug(fName, "starting")
	var (
		ss     *gamestate.SolarSystem
		obj    *gamestate.SolarObject
		exists bool
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.Owner == 0 {
				continue
			}
			_, exists = galaxy.PlayerFogOfWar[obj.Owner]
			if !exists {
				galaxy.PlayerFogOfWar[obj.Owner] = map[uint64]*gamestate.SolarSystem{}
				galaxy.PlayerFogOfWar[obj.Owner][ss.NameUint] = ss
			} else if galaxy.PlayerFogOfWar[obj.Owner][ss.NameUint] != ss {
				galaxy.PlayerFogOfWar[obj.Owner][ss.NameUint] = ss
			}
		}
	}
}
