package backend

import "sync"

// PointerMappings ...
type PointerMappings struct {
	solarObjectMappings map[uint64]*SolarObject
	solarSystemMappings map[uint64]*SolarSystem

	galacticFleetMutex    *sync.RWMutex
	galacticFleetMappings map[int32]*GalacticFleet
	galacticFleetSlice    []*GalacticFleet

	ongoingBattleMutex    *sync.RWMutex
	ongoingBattleMappings map[int32]*OngoingBattle
	ongoingBattleSlice    []*OngoingBattle

	ongoingSiegeMutex    *sync.RWMutex
	ongoingSiegeMappings map[int32]*OngoingSiege
	ongoingSiegeSlice    []*OngoingSiege

	outstandingActionMutex    *sync.RWMutex
	outstandingActionMappings map[int32]*OutstandingAction
	outstandingActionSlice    []*OutstandingAction
}

// NewPointerMappings ...
func NewPointerMappings() *PointerMappings {
	ptrMappings := &PointerMappings{}

	ptrMappings.solarObjectMappings = map[uint64]*SolarObject{}
	ptrMappings.solarSystemMappings = map[uint64]*SolarSystem{}

	ptrMappings.galacticFleetMutex = &sync.RWMutex{}
	ptrMappings.galacticFleetMappings = map[int32]*GalacticFleet{}

	ptrMappings.ongoingBattleMutex = &sync.RWMutex{}
	ptrMappings.ongoingBattleMappings = map[int32]*OngoingBattle{}

	ptrMappings.ongoingSiegeMutex = &sync.RWMutex{}
	ptrMappings.ongoingSiegeMappings = map[int32]*OngoingSiege{}

	ptrMappings.outstandingActionMutex = &sync.RWMutex{}
	ptrMappings.outstandingActionMappings = map[int32]*OutstandingAction{}

	return ptrMappings
}

// GetSolarObjectFromName ...
func (galaxy *GalaxyStruct) GetSolarObjectFromName(objNameInt uint64) *SolarObject {
	return galaxy.PointerMappings.solarObjectMappings[objNameInt]
}

// GetSolarSystemFromName ...
func (galaxy *GalaxyStruct) GetSolarSystemFromName(sysNameInt uint64) *SolarSystem {
	return galaxy.PointerMappings.solarSystemMappings[sysNameInt]
}

// GetFleetFromID ...
func (galaxy *GalaxyStruct) GetFleetFromID(fleetID int32) *GalacticFleet {
	galaxy.PointerMappings.galacticFleetMutex.RLock()
	defer galaxy.PointerMappings.galacticFleetMutex.RUnlock()
	return galaxy.Protobuf.Galacticfleetslice[int(fleetID)]
}

// GetBattleFromID ...
func (galaxy *GalaxyStruct) GetBattleFromID(battleID int32) *OngoingBattle {
	galaxy.PointerMappings.ongoingBattleMutex.RLock()
	defer galaxy.PointerMappings.ongoingBattleMutex.RUnlock()
	return galaxy.Protobuf.Ongoingbattleslice[int(battleID)]
}

// GetSiegeFromID ...
func (galaxy *GalaxyStruct) GetSiegeFromID(siegeID int32) *OngoingSiege {
	galaxy.PointerMappings.ongoingSiegeMutex.RLock()
	defer galaxy.PointerMappings.ongoingSiegeMutex.RUnlock()
	return galaxy.Protobuf.Ongoingsiegeslice[int(siegeID)]
}

// GetActionFromID ...
func (galaxy *GalaxyStruct) GetActionFromID(actionID int32) *OutstandingAction {
	galaxy.PointerMappings.outstandingActionMutex.RLock()
	defer galaxy.PointerMappings.outstandingActionMutex.RUnlock()
	return galaxy.Protobuf.Outstandingactionslice[int(actionID)]
}
