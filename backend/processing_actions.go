package backend

import (
	"math"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/vectors"
)

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
	galaxy.Protobuf.Actionslice = make([]*gamestate.Action, 0)
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

// DistanceBetweenPoints is a convienence function around the hypotenuse function
func DistanceBetweenPoints(x1, y1, x2, y2 int64) uint64 {
	return uint64(math.Hypot(float64(x1)-float64(x2), float64(y1)-float64(y2)))
}

const (
	fleetSpeedPerSecond     = lightYear / 1000
	maximumFleetAttackRange = 20 * fleetSpeedPerSecond
	optimalFleetAttackRange = 10 * fleetSpeedPerSecond
)

// ExecuteFleetAction ...
func ExecuteFleetAction(action *gamestate.Action, galaxy *GalaxyStruct) {
	switch action.ActionType {
	case gamestate.MoveFleetToAttackFleet:
		// BUG: are locks needed?
		myFleet := galaxy.PointerMappings.fleetMappings[int32(action.ActionValues[0])]
		otherFleet := galaxy.PointerMappings.fleetMappings[int32(action.ActionValues[1])]
		distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
			otherFleet.InterX, otherFleet.InterY)
		if distance > optimalFleetAttackRange {
			myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
				myFleet.InterX, myFleet.InterY, otherFleet.InterY, otherFleet.InterY, fleetSpeedPerSecond)
		}
	case gamestate.MoveFleetToAttackSystem:
		myFleet := galaxy.PointerMappings.fleetMappings[int32(action.ActionValues[0])]
		otherSystem := galaxy.PointerMappings.systemMappings[action.ActionNames[0]]
		distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
			otherSystem.InterX, otherSystem.InterY)
		if distance > lightYear/100 {
			myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
				myFleet.InterX, myFleet.InterY, otherSystem.InterY, otherSystem.InterY, fleetSpeedPerSecond)
		}
	case gamestate.MoveFleetToDefendSystem:
		myFleet := galaxy.PointerMappings.fleetMappings[int32(action.ActionValues[0])]
		mySystem := galaxy.PointerMappings.systemMappings[action.ActionNames[0]]
		distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
			mySystem.InterX, mySystem.InterY)
		if distance > lightYear/100 {
			myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
				myFleet.InterX, myFleet.InterY, mySystem.InterY, mySystem.InterY, fleetSpeedPerSecond)
		}
	case gamestate.MoveFleetToOngoingBattle:
		myFleet := galaxy.PointerMappings.fleetMappings[int32(action.ActionValues[0])]
		battle := galaxy.PointerMappings.battleMappings[int32(action.ActionValues[1])]
		originFleet := galaxy.PointerMappings.fleetMappings[battle.GetBattleOriginFleetID()]
		distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
			originFleet.InterX, originFleet.InterY)
		if distance > optimalFleetAttackRange {
			myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
				myFleet.InterX, myFleet.InterY, originFleet.InterY, originFleet.InterY, fleetSpeedPerSecond)
		}
	}
}

// ExecutePlanetAction ...
func ExecutePlanetAction(action *gamestate.Action, galaxy *GalaxyStruct) {

}
