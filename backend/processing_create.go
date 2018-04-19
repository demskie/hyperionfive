package backend

// CreateGalacticFleet ...
func (galaxy *GalaxyStruct) CreateGalacticFleet(fleet *GalacticFleet) {
	galaxy.PointerMappings.galacticFleetMutex.Lock()
	defer galaxy.PointerMappings.galacticFleetMutex.Unlock()
	fleet.FleetID = int32(len(galaxy.Protobuf.Galacticfleetslice))
	galaxy.Protobuf.Galacticfleetslice = append(galaxy.Protobuf.Galacticfleetslice, fleet)
}

// CreateOngoingBattle ...
func (galaxy *GalaxyStruct) CreateOngoingBattle(battle *OngoingBattle) {
	galaxy.PointerMappings.ongoingBattleMutex.Lock()
	defer galaxy.PointerMappings.ongoingBattleMutex.Unlock()
	battle.BattleID = int32(len(galaxy.Protobuf.Ongoingbattleslice))
	galaxy.Protobuf.Ongoingbattleslice = append(galaxy.Protobuf.Ongoingbattleslice, battle)
}

// CreateOngoingSiege ...
func (galaxy *GalaxyStruct) CreateOngoingSiege(siege *OngoingSiege) {
	galaxy.PointerMappings.ongoingSiegeMutex.Lock()
	defer galaxy.PointerMappings.ongoingSiegeMutex.Unlock()
	siege.SiegeID = int32(len(galaxy.Protobuf.Ongoingsiegeslice))
	galaxy.Protobuf.Ongoingsiegeslice = append(galaxy.Protobuf.Ongoingsiegeslice, siege)
}

// CreateOutstandingAction ...
func (galaxy *GalaxyStruct) CreateOutstandingAction(action *OutstandingAction) {
	galaxy.PointerMappings.outstandingActionMutex.Lock()
	defer galaxy.PointerMappings.outstandingActionMutex.Unlock()
	action.ActionID = int32(len(galaxy.Protobuf.Outstandingactionslice))
	galaxy.Protobuf.Outstandingactionslice = append(galaxy.Protobuf.Outstandingactionslice, action)
}
