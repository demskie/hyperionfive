package main

// PreprocessPointerMappings ...
func (galaxy *GalaxyStruct) PreprocessPointerMappings() {
	const fName = "(preprocessPointerMappings)"
	logDebug(fName, "starting")
	var (
		ss     *SolarSystem
		obj    *SolarObject
		fleet  *GalacticFleet
		battle *OngoingBattle
		siege  *OngoingSiege
		action *OutstandingAction
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			galaxy.PointerMappings.solarObjectMappings[obj.Name] = obj
		}
		galaxy.PointerMappings.solarSystemMappings[ss.Name] = ss
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
	const fName = "(preprocessAllPlanetPointers)"
	logDebug(fName, "starting")
	var (
		ss  *SolarSystem
		obj *SolarObject
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.IsPlanet() {
				galaxy.PopulationObjectsSlice = append(galaxy.PopulationObjectsSlice, obj)
				if obj.Owner == NEUTRAL && obj.Type == TERRAN {
					galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, obj)
				}
			}
		}
	}
}

// PreprocessPlayerFogOfWar ...
func (galaxy *GalaxyStruct) PreprocessPlayerFogOfWar() {
	const fName = "(preprocessPlayerFogOfWar)"
	logDebug(fName, "starting")
	var (
		ss     *SolarSystem
		obj    *SolarObject
		exists bool
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.Owner == NEUTRAL {
				continue
			}
			_, exists = galaxy.PlayerFogOfWar[obj.Owner]
			if !exists {
				galaxy.PlayerFogOfWar[obj.Owner] = map[string]*SolarSystem{}
				galaxy.PlayerFogOfWar[obj.Owner][ss.Name] = ss
			} else if galaxy.PlayerFogOfWar[obj.Owner][ss.Name] != ss {
				galaxy.PlayerFogOfWar[obj.Owner][ss.Name] = ss
			}
		}
	}
}
