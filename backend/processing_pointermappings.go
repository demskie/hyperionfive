package backend

import (
	"sync"

	"github.com/demskie/hyperionfive/gamestate"
)

// PointerMappings ...
type PointerMappings struct {
	solarObjectMappings map[uint64]*gamestate.SolarObject
	solarSystemMappings map[uint64]*gamestate.SolarSystem

	galacticFleetMutex    *sync.RWMutex
	galacticFleetMappings map[int32]*gamestate.GalacticFleet
	galacticFleetSlice    []*gamestate.GalacticFleet

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

	ptrMappings.solarObjectMappings = map[uint64]*gamestate.SolarObject{}
	ptrMappings.solarSystemMappings = map[uint64]*gamestate.SolarSystem{}

	ptrMappings.galacticFleetMutex = &sync.RWMutex{}
	ptrMappings.galacticFleetMappings = map[int32]*gamestate.GalacticFleet{}

	ptrMappings.ongoingBattleMutex = &sync.RWMutex{}
	ptrMappings.ongoingBattleMappings = map[int32]*gamestate.OngoingBattle{}

	ptrMappings.ongoingSiegeMutex = &sync.RWMutex{}
	ptrMappings.ongoingSiegeMappings = map[int32]*gamestate.OngoingSiege{}

	ptrMappings.outstandingActionMutex = &sync.RWMutex{}
	ptrMappings.outstandingActionMappings = map[int32]*gamestate.OutstandingAction{}

	return ptrMappings
}

// GetSolarObjectFromName ...
func (galaxy *GalaxyStruct) GetSolarObjectFromName(objNameInt uint64) *gamestate.SolarObject {
	return galaxy.PointerMappings.solarObjectMappings[objNameInt]
}

// GetSolarSystemFromName ...
func (galaxy *GalaxyStruct) GetSolarSystemFromName(sysNameInt uint64) *gamestate.SolarSystem {
	return galaxy.PointerMappings.solarSystemMappings[sysNameInt]
}

// GetFleetFromID ...
func (galaxy *GalaxyStruct) GetFleetFromID(fleetID int32) *gamestate.GalacticFleet {
	galaxy.PointerMappings.galacticFleetMutex.RLock()
	defer galaxy.PointerMappings.galacticFleetMutex.RUnlock()
	return galaxy.Protobuf.Galacticfleetslice[int(fleetID)]
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
