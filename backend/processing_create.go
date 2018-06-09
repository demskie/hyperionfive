package backend

import "github.com/demskie/hyperionfive/gamestate"

// CreateFleet ...
func (galaxy *GalaxyStruct) CreateFleet(fleet *gamestate.Fleet) {
	galaxy.PointerMappings.fleetMutex.Lock()
	defer galaxy.PointerMappings.fleetMutex.Unlock()
	fleet.FleetID = int32(len(galaxy.Protobuf.Fleetslice))
	galaxy.Protobuf.Fleetslice = append(galaxy.Protobuf.Fleetslice, fleet)
}

// CreateOngoingBattle ...
func (galaxy *GalaxyStruct) CreateOngoingBattle(battle *gamestate.OngoingBattle) {
	galaxy.PointerMappings.ongoingBattleMutex.Lock()
	defer galaxy.PointerMappings.ongoingBattleMutex.Unlock()
	battle.BattleID = int32(len(galaxy.Protobuf.Ongoingbattleslice))
	galaxy.Protobuf.Ongoingbattleslice = append(galaxy.Protobuf.Ongoingbattleslice, battle)
}

// CreateOngoingSiege ...
func (galaxy *GalaxyStruct) CreateOngoingSiege(siege *gamestate.OngoingSiege) {
	galaxy.PointerMappings.ongoingSiegeMutex.Lock()
	defer galaxy.PointerMappings.ongoingSiegeMutex.Unlock()
	siege.SiegeID = int32(len(galaxy.Protobuf.Ongoingsiegeslice))
	galaxy.Protobuf.Ongoingsiegeslice = append(galaxy.Protobuf.Ongoingsiegeslice, siege)
}

// CreateOutstandingAction ...
func (galaxy *GalaxyStruct) CreateOutstandingAction(action *gamestate.OutstandingAction) {
	galaxy.PointerMappings.outstandingActionMutex.Lock()
	defer galaxy.PointerMappings.outstandingActionMutex.Unlock()
	action.ActionID = int32(len(galaxy.Protobuf.Outstandingactionslice))
	galaxy.Protobuf.Outstandingactionslice = append(galaxy.Protobuf.Outstandingactionslice, action)
}
