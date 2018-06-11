package backend

import (
	"sync"

	"github.com/demskie/hyperionfive/gamestate"
)

// PointerMappings ...
type PointerMappings struct {
	planetMappings map[uint64]*gamestate.Planet

	objectMutex    *sync.RWMutex
	objectMappings map[uint64]*gamestate.Object

	systemMappings map[uint64]*gamestate.System

	fleetMutex    *sync.RWMutex
	fleetMappings map[int32]*gamestate.Fleet

	battleMutex    *sync.RWMutex
	battleMappings map[int32]*gamestate.Battle

	siegeMutex    *sync.RWMutex
	siegeMappings map[int32]*gamestate.Siege

	actionMutex    *sync.RWMutex
	actionMappings map[int32]*gamestate.Action
}

// NewPointerMappings ...
func NewPointerMappings() *PointerMappings {
	ptrMappings := &PointerMappings{}

	ptrMappings.planetMappings = map[uint64]*gamestate.Planet{}

	ptrMappings.objectMutex = &sync.RWMutex{}
	ptrMappings.objectMappings = map[uint64]*gamestate.Object{}

	ptrMappings.systemMappings = map[uint64]*gamestate.System{}

	ptrMappings.fleetMutex = &sync.RWMutex{}
	ptrMappings.fleetMappings = map[int32]*gamestate.Fleet{}

	ptrMappings.battleMutex = &sync.RWMutex{}
	ptrMappings.battleMappings = map[int32]*gamestate.Battle{}

	ptrMappings.siegeMutex = &sync.RWMutex{}
	ptrMappings.siegeMappings = map[int32]*gamestate.Siege{}

	ptrMappings.actionMutex = &sync.RWMutex{}
	ptrMappings.actionMappings = map[int32]*gamestate.Action{}

	return ptrMappings
}

// GetPlanetFromName ...
func (galaxy *GalaxyStruct) GetPlanetFromName(planetNameUint uint64) *gamestate.Planet {
	return galaxy.PointerMappings.planetMappings[planetNameUint]
}

// GetObjectFromName ...
func (galaxy *GalaxyStruct) GetObjectFromName(objectNameUint uint64) *gamestate.Object {
	galaxy.PointerMappings.objectMutex.RLock()
	defer galaxy.PointerMappings.objectMutex.RUnlock()
	return galaxy.PointerMappings.objectMappings[objectNameUint]
}

// GetSystemFromName ...
func (galaxy *GalaxyStruct) GetSystemFromName(sysNameUint uint64) *gamestate.System {
	return galaxy.PointerMappings.systemMappings[sysNameUint]
}

// GetFleetFromID ...
func (galaxy *GalaxyStruct) GetFleetFromID(fleetID int32) *gamestate.Fleet {
	galaxy.PointerMappings.fleetMutex.RLock()
	defer galaxy.PointerMappings.fleetMutex.RUnlock()
	return galaxy.Protobuf.Fleetslice[int(fleetID)]
}

// GetBattleFromID ...
func (galaxy *GalaxyStruct) GetBattleFromID(battleID int32) *gamestate.Battle {
	galaxy.PointerMappings.battleMutex.RLock()
	defer galaxy.PointerMappings.battleMutex.RUnlock()
	return galaxy.Protobuf.Battleslice[int(battleID)]
}

// GetSiegeFromID ...
func (galaxy *GalaxyStruct) GetSiegeFromID(siegeID int32) *gamestate.Siege {
	galaxy.PointerMappings.siegeMutex.RLock()
	defer galaxy.PointerMappings.siegeMutex.RUnlock()
	return galaxy.Protobuf.Siegeslice[int(siegeID)]
}

// GetActionFromID ...
func (galaxy *GalaxyStruct) GetActionFromID(actionID int32) *gamestate.Action {
	galaxy.PointerMappings.actionMutex.RLock()
	defer galaxy.PointerMappings.actionMutex.RUnlock()
	return galaxy.Protobuf.Actionslice[int(actionID)]
}
