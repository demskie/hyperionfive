package backend

import (
	"math"

	"github.com/demskie/hyperionfive/backendmessage"
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
		case backendmessage.MoveFleetToAttackFleet:
			var myFleet, otherFleet *backendmessage.Fleet
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				otherFleet.InterX, otherFleet.InterY)
			if distance <= backendmessage.OptimalFleetAttackRange {
				// create battle
				delete(galaxy.Protobuf.Action, actionID)
			} else if distance > backendmessage.OptimalFleetAttackRange {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					otherFleet.InterY, otherFleet.InterY,
					backendmessage.FleetSpeedPerSecond)
			}
		case backendmessage.MoveFleetToAttackSystem:
			var myFleet *backendmessage.Fleet
			var otherSystem *backendmessage.System
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				otherSystem.InterX, otherSystem.InterY)
			if distance > backendmessage.LightYear/100 {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					otherSystem.InterY, otherSystem.InterY,
					backendmessage.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		case backendmessage.MoveFleetToDefendSystem:
			var myFleet *backendmessage.Fleet
			var mySystem *backendmessage.System
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				mySystem.InterX, mySystem.InterY)
			if distance > backendmessage.LightYear/100 {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					mySystem.InterY, mySystem.InterY,
					backendmessage.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		case backendmessage.MoveFleetToOngoingBattle:
			var myFleet *backendmessage.Fleet
			var originFleet *backendmessage.Fleet
			distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
				originFleet.InterX, originFleet.InterY)
			if distance > backendmessage.OptimalFleetAttackRange {
				myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
					myFleet.InterX, myFleet.InterY,
					originFleet.InterY, originFleet.InterY,
					backendmessage.FleetSpeedPerSecond)
			} else {
				delete(galaxy.Protobuf.Action, actionID)
			}
		}
	}
}
