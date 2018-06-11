package backend

import "github.com/demskie/hyperionfive/gamestate"

// BUG // need to sweep through and cleanup unused IDs

// CreateFleet ...
func (galaxy *GalaxyStruct) CreateFleet(fleet *gamestate.Fleet) {
	galaxy.PointerMappings.fleetMutex.Lock()
	fleet.FleetID = galaxy.FleetIDs.Create()
	galaxy.Protobuf.Fleetslice = append(galaxy.Protobuf.Fleetslice, fleet)
	galaxy.PointerMappings.fleetMappings[fleet.FleetID] = fleet
	galaxy.PointerMappings.fleetMutex.Unlock()
}

// CreateOngoingBattle ...
func (galaxy *GalaxyStruct) CreateOngoingBattle(battle *gamestate.Battle) {
	galaxy.PointerMappings.battleMutex.Lock()
	battle.BattleID = galaxy.BattleIDs.Create()
	galaxy.Protobuf.Battleslice = append(galaxy.Protobuf.Battleslice, battle)
	galaxy.PointerMappings.battleMappings[battle.BattleID] = battle
	galaxy.PointerMappings.battleMutex.Unlock()
}

// CreateOngoingSiege ...
func (galaxy *GalaxyStruct) CreateOngoingSiege(siege *gamestate.Siege) {
	galaxy.PointerMappings.siegeMutex.Lock()
	siege.SiegeID = galaxy.SiegeIDs.Create()
	galaxy.Protobuf.Siegeslice = append(galaxy.Protobuf.Siegeslice, siege)
	galaxy.PointerMappings.siegeMappings[siege.SiegeID] = siege
	galaxy.PointerMappings.siegeMutex.Unlock()
}

// CreateOutstandingAction ...
func (galaxy *GalaxyStruct) CreateOutstandingAction(action *gamestate.Action) {
	galaxy.PointerMappings.actionMutex.Lock()
	action.ActionID = galaxy.ActionIDs.Create()
	galaxy.Protobuf.Actionslice = append(galaxy.Protobuf.Actionslice, action)
	galaxy.PointerMappings.actionMappings[action.ActionID] = action
	galaxy.PointerMappings.actionMutex.Unlock()
}
