package backend

import "github.com/demskie/hyperionfive/gamestate"

// LookupPlanetName ...
func (galaxy *GalaxyStruct) LookupPlanetName(planetNameUint uint64) *gamestate.Planet {
	return galaxy.Protobuf.Planetmap[planetNameUint]
}

// LookupObjectName ...
func (galaxy *GalaxyStruct) LookupObjectName(objectNameUint uint64) *gamestate.Object {
	galaxy.ObjectMapMutex.RLock()
	val := galaxy.Protobuf.Objectmap[objectNameUint]
	galaxy.ObjectMapMutex.RUnlock()
	return val
}

// LookupSystemName ...
func (galaxy *GalaxyStruct) LookupSystemName(sysNameUint uint64) *gamestate.System {
	return galaxy.Protobuf.Systemmap[sysNameUint]
}

// LookupFleetID ...
func (galaxy *GalaxyStruct) LookupFleetID(fleetID int32) *gamestate.Fleet {
	galaxy.FleetMapMutex.RLock()
	val := galaxy.Protobuf.Fleetmap[fleetID]
	galaxy.FleetMapMutex.RUnlock()
	return val
}

// LookupBattleID ...
func (galaxy *GalaxyStruct) LookupBattleID(battleID int32) *gamestate.Battle {
	galaxy.BattleMapMutex.RLock()
	val := galaxy.Protobuf.Battlemap[battleID]
	galaxy.BattleMapMutex.RUnlock()
	return val
}

// LookupSiegeID ...
func (galaxy *GalaxyStruct) LookupSiegeID(siegeID int32) *gamestate.Siege {
	galaxy.SiegeMapMutex.RLock()
	val := galaxy.Protobuf.Siegemap[siegeID]
	galaxy.SiegeMapMutex.RUnlock()
	return val
}

// LookupActionID ...
func (galaxy *GalaxyStruct) LookupActionID(actionID int32) *gamestate.Action {
	galaxy.ActionMapMutex.RLock()
	val := galaxy.Protobuf.Actionmap[actionID]
	galaxy.ActionMapMutex.RUnlock()
	return val
}
