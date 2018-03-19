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

// GetFleetFromID ...
func (galaxy *GalaxyStruct) GetFleetFromID(fleetID int32) (fleet *GalacticFleet) {
	return galaxy.PointerMappings.galacticFleetSlice[int(fleetID)]
}

// GetBattleFromID ...
func (galaxy *GalaxyStruct) GetBattleFromID(battleID int32) (battle *OngoingBattle) {
	return galaxy.PointerMappings.ongoingBattleSlice[int(battleID)]
}
