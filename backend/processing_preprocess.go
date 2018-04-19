package backend

// PreprocessPointerMappings ...
func (galaxy *GalaxyStruct) PreprocessPointerMappings() {
	const fName = "(preprocessPointerMappings)"
	log.Debug(fName, "starting")
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
			galaxy.PointerMappings.solarObjectMappings[obj.NameInt] = obj
		}
		galaxy.PointerMappings.solarSystemMappings[ss.NameInt] = ss
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
		ss  *SolarSystem
		obj *SolarObject
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.IsPlanet() {
				galaxy.AnyPlanetSlice = append(galaxy.AnyPlanetSlice, obj)
				if obj.Owner == 0 {
					galaxy.NeutralAnyPlanetSlice = append(galaxy.NeutralAnyPlanetSlice, obj)
				}
				if obj.Owner == 0 && obj.Type == TERRAN {
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
		ss     *SolarSystem
		obj    *SolarObject
		exists bool
	)
	for _, ss = range galaxy.Protobuf.Solarsystemslice {
		for _, obj = range ss.Solarobjectslice {
			if obj.Owner == 0 {
				continue
			}
			_, exists = galaxy.PlayerFogOfWar[obj.Owner]
			if !exists {
				galaxy.PlayerFogOfWar[obj.Owner] = map[uint64]*SolarSystem{}
				galaxy.PlayerFogOfWar[obj.Owner][ss.NameInt] = ss
			} else if galaxy.PlayerFogOfWar[obj.Owner][ss.NameInt] != ss {
				galaxy.PlayerFogOfWar[obj.Owner][ss.NameInt] = ss
			}
		}
	}
}
