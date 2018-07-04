package backend

import "github.com/demskie/hyperionfive/gamestate"

// BUG // need to sweep through and cleanup unused IDs

// CreateFleet ...
func (galaxy *GalaxyStruct) CreateFleet(fleet *gamestate.Fleet) {
	galaxy.FleetMapMutex.Lock()
	fleet.FleetID = galaxy.FleetIDs.Create()
	galaxy.Protobuf.Fleetslice = append(galaxy.Protobuf.Fleetslice, fleet)
	galaxy.Protobuf.Fleetmap[fleet.FleetID] = fleet
	galaxy.FleetMapMutex.Unlock()
}

// CreateOngoingBattle ...
func (galaxy *GalaxyStruct) CreateOngoingBattle(battle *gamestate.Battle) {
	galaxy.BattleMapMutex.Lock()
	battle.BattleID = galaxy.BattleIDs.Create()
	galaxy.Protobuf.Battleslice = append(galaxy.Protobuf.Battleslice, battle)
	galaxy.Protobuf.Battlemap[battle.BattleID] = battle
	galaxy.BattleMapMutex.Unlock()
}

// CreateOngoingSiege ...
func (galaxy *GalaxyStruct) CreateOngoingSiege(siege *gamestate.Siege) {
	galaxy.SiegeMapMutex.Lock()
	siege.SiegeID = galaxy.SiegeIDs.Create()
	galaxy.Protobuf.Siegeslice = append(galaxy.Protobuf.Siegeslice, siege)
	galaxy.Protobuf.Siegemap[siege.SiegeID] = siege
	galaxy.SiegeMapMutex.Unlock()
}

// CreateOutstandingAction ...
func (galaxy *GalaxyStruct) CreateOutstandingAction(action *gamestate.Action) {
	galaxy.ActionMapMutex.Lock()
	action.ActionID = galaxy.ActionIDs.Create()
	galaxy.Protobuf.Actionslice = append(galaxy.Protobuf.Actionslice, action)
	galaxy.Protobuf.Actionmap[action.ActionID] = action
	galaxy.ActionMapMutex.Unlock()
}
