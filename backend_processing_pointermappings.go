package main

import "sync"

type pointerMappings struct {
	solarObjectMappings map[string]*SolarObject
	solarSystemMappings map[string]*SolarSystem

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

func (ptrMappings *pointerMappings) Initialize() {
	ptrMappings.solarObjectMappings = map[string]*SolarObject{}
	ptrMappings.solarSystemMappings = map[string]*SolarSystem{}

	ptrMappings.galacticFleetMutex = &sync.RWMutex{}
	ptrMappings.galacticFleetMappings = map[int32]*GalacticFleet{}
	ptrMappings.galacticFleetSlice = []*GalacticFleet{}

	ptrMappings.ongoingSiegeMutex = &sync.RWMutex{}
	ptrMappings.ongoingBattleMappings = map[int32]*OngoingBattle{}
	ptrMappings.ongoingBattleSlice = []*OngoingBattle{}

	ptrMappings.ongoingSiegeMutex = &sync.RWMutex{}
	ptrMappings.ongoingSiegeMappings = map[int32]*OngoingSiege{}
	ptrMappings.ongoingSiegeSlice = []*OngoingSiege{}

	ptrMappings.outstandingActionMutex = &sync.RWMutex{}
	ptrMappings.outstandingActionMappings = map[int32]*OutstandingAction{}
	ptrMappings.outstandingActionSlice = []*OutstandingAction{}
}

// GetSolarObjectFromName ...
func (galaxy *GalaxyStruct) GetSolarObjectFromName(objName string) *SolarObject {
	return galaxy.PointerMappings.solarObjectMappings[objName]
}

// GetSolarSystemFromName ...
func (galaxy *GalaxyStruct) GetSolarSystemFromName(sysName string) *SolarSystem {
	return galaxy.PointerMappings.solarSystemMappings[sysName]
}

// GetFleetFromID ...
func (galaxy *GalaxyStruct) GetFleetFromID(fleetID int32) *GalacticFleet {
	galaxy.PointerMappings.galacticFleetMutex.RLock()
	defer galaxy.PointerMappings.galacticFleetMutex.RUnlock()
	return galaxy.PointerMappings.galacticFleetSlice[int(fleetID)]
}

// GetBattleFromID ...
func (galaxy *GalaxyStruct) GetBattleFromID(battleID int32) *OngoingBattle {
	galaxy.PointerMappings.ongoingBattleMutex.RLock()
	defer galaxy.PointerMappings.ongoingBattleMutex.RUnlock()
	return galaxy.PointerMappings.ongoingBattleSlice[int(battleID)]
}

// GetSiegeFromID ...
func (galaxy *GalaxyStruct) GetSiegeFromID(siegeID int32) *OngoingSiege {
	galaxy.PointerMappings.ongoingSiegeMutex.RLock()
	defer galaxy.PointerMappings.ongoingSiegeMutex.RUnlock()
	return galaxy.PointerMappings.ongoingSiegeSlice[int(siegeID)]
}

// GetActionFromID ...
func (galaxy *GalaxyStruct) GetActionFromID(actionID int32) *OutstandingAction {
	galaxy.PointerMappings.outstandingActionMutex.RLock()
	defer galaxy.PointerMappings.outstandingActionMutex.RUnlock()
	return galaxy.PointerMappings.outstandingActionSlice[int(actionID)]
}
