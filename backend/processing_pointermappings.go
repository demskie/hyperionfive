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
	fleetSlice    []*gamestate.Fleet

	ongoingBattleMutex    *sync.RWMutex
	ongoingBattleMappings map[int32]*gamestate.OngoingBattle
	ongoingBattleSlice    []*gamestate.OngoingBattle

	ongoingSiegeMutex    *sync.RWMutex
	ongoingSiegeMappings map[int32]*gamestate.OngoingSiege
	ongoingSiegeSlice    []*gamestate.OngoingSiege

	outstandingActionMutex    *sync.RWMutex
	outstandingActionMappings map[int32]*gamestate.OutstandingAction
	outstandingActionSlice    []*gamestate.OutstandingAction
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
	ptrMappings.fleetSlice = []*gamestate.Fleet{}

	ptrMappings.ongoingBattleMutex = &sync.RWMutex{}
	ptrMappings.ongoingBattleMappings = map[int32]*gamestate.OngoingBattle{}
	ptrMappings.ongoingBattleSlice = []*gamestate.OngoingBattle{}

	ptrMappings.ongoingSiegeMutex = &sync.RWMutex{}
	ptrMappings.ongoingSiegeMappings = map[int32]*gamestate.OngoingSiege{}
	ptrMappings.ongoingSiegeSlice = []*gamestate.OngoingSiege{}

	ptrMappings.outstandingActionMutex = &sync.RWMutex{}
	ptrMappings.outstandingActionMappings = map[int32]*gamestate.OutstandingAction{}
	ptrMappings.outstandingActionSlice = []*gamestate.OutstandingAction{}

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
func (galaxy *GalaxyStruct) GetBattleFromID(battleID int32) *gamestate.OngoingBattle {
	galaxy.PointerMappings.ongoingBattleMutex.RLock()
	defer galaxy.PointerMappings.ongoingBattleMutex.RUnlock()
	return galaxy.Protobuf.Ongoingbattleslice[int(battleID)]
}

// GetSiegeFromID ...
func (galaxy *GalaxyStruct) GetSiegeFromID(siegeID int32) *gamestate.OngoingSiege {
	galaxy.PointerMappings.ongoingSiegeMutex.RLock()
	defer galaxy.PointerMappings.ongoingSiegeMutex.RUnlock()
	return galaxy.Protobuf.Ongoingsiegeslice[int(siegeID)]
}

// GetActionFromID ...
func (galaxy *GalaxyStruct) GetActionFromID(actionID int32) *gamestate.OutstandingAction {
	galaxy.PointerMappings.outstandingActionMutex.RLock()
	defer galaxy.PointerMappings.outstandingActionMutex.RUnlock()
	return galaxy.Protobuf.Outstandingactionslice[int(actionID)]
}
