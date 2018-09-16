package backend

import (
	"math"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/vectors"
)

// DistanceBetweenPoints is a convienence function around the hypotenuse function
func DistanceBetweenPoints(x1, y1, x2, y2 int64) uint64 {
	return uint64(math.Hypot(float64(x1)-float64(x2), float64(y1)-float64(y2)))
}

// ProcessActions ...
func (galaxy *GalaxyStruct) ProcessActions() {
	const fName = "(processActions)"
	if len(galaxy.Protobuf.Action) == 0 {
		return
	}
	for actionID, action := range galaxy.Protobuf.Action {
		switch action.ActionType {
		case gamestate.MoveFleetToAttackFleet:
			var myFleet, otherFleet *gamestate.Fleet
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				otherFleet.InterX, otherFleet.InterY)
			if distance <= gamestate.OptimalFleetAttackRange {
				// create battle
				delete(galaxy.Protobuf.Action, actionID)
			} else if distance > gamestate.OptimalFleetAttackRange {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					otherFleet.InterY, otherFleet.InterY,
					gamestate.FleetSpeedPerSecond)
			}
		case gamestate.MoveFleetToAttackSystem:
			var myFleet *gamestate.Fleet
			var otherSystem *gamestate.System
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				otherSystem.InterX, otherSystem.InterY)
			if distance > gamestate.LightYear/100 {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					otherSystem.InterY, otherSystem.InterY,
					gamestate.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		case gamestate.MoveFleetToDefendSystem:
			var myFleet *gamestate.Fleet
			var mySystem *gamestate.System
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				mySystem.InterX, mySystem.InterY)
			if distance > gamestate.LightYear/100 {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					mySystem.InterY, mySystem.InterY,
					gamestate.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		case gamestate.MoveFleetToOngoingBattle:
			var myFleet *gamestate.Fleet
			var originFleet *gamestate.Fleet
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				originFleet.InterX, originFleet.InterY)
			if distance > gamestate.OptimalFleetAttackRange {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					originFleet.InterY, originFleet.InterY,
					gamestate.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		}
	}
}
