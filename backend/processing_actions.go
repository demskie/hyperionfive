package backend

import "github.com/demskie/hyperionfive/gamestate"

// ProcessActions ...
func (galaxy *GalaxyStruct) ProcessActions() {
	const fName = "(processActions)"
	if len(galaxy.Protobuf.Actionslice) == 0 {
		return
	}
	var (
		fleetAction  = map[int64]*gamestate.Action{}
		undockAction = map[uint64]*gamestate.Action{}
	)
	for _, action := range galaxy.Protobuf.Actionslice {
		switch action.ActionType {
		case gamestate.MoveFleetToAttackFleet:
			fleetAction[action.ActionValues[0]] = action
		case gamestate.MoveFleetToAttackSystem:
			fleetAction[action.ActionValues[0]] = action
		case gamestate.MoveFleetToDefendSystem:
			fleetAction[action.ActionValues[0]] = action
		case gamestate.MoveFleetToOngoingBattle:
			fleetAction[action.ActionValues[0]] = action
		case gamestate.UndockFleetFromPlanet:
			undockAction[action.ActionNames[0]] = action
		}
	}
	semaphore := make(chan struct{}, args.NumberOfCores)
	for _, action := range fleetAction {
		<-semaphore
		go func(anAction *gamestate.Action) {
			ExecuteFleetAction(anAction, galaxy)
			semaphore <- struct{}{}
		}(action)
	}
	for _, action := range undockAction {
		<-semaphore
		go func(anAction *gamestate.Action) {
			ExecutePlanetAction(anAction, galaxy)
			semaphore <- struct{}{}
		}(action)
	}
}

// ExecuteFleetAction ...
func ExecuteFleetAction(action *gamestate.Action, galaxy *GalaxyStruct) {

}

// ExecutePlanetAction ...
func ExecutePlanetAction(action *gamestate.Action, galaxy *GalaxyStruct) {

}
