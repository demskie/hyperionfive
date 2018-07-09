package backend

import (
	"math"
	"sync"

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
	wg := sync.WaitGroup{}
	wg.Add(1)
	actionChan := make(chan *gamestate.Action, 1000)
	futureActionChan := make(chan *gamestate.Action, 1000)
	go func() {
		for _, action := range fleetAction {
			actionChan <- action
		}
		for _, action := range undockAction {
			actionChan <- action
		}
		close(actionChan)
		var i int
		for futureAction := range futureActionChan {
			galaxy.Protobuf.Actionslice[i] = futureAction
			i++
		}
		galaxy.Protobuf.Actionslice = galaxy.Protobuf.Actionslice[:i]
		wg.Done()
	}()
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		for action := range actionChan {
			switch action.ActionType {
			case gamestate.MoveFleetToAttackFleet:
				myFleet := galaxy.LookupFleetID(int32(action.ActionValues[0]))
				otherFleet := galaxy.LookupFleetID(int32(action.ActionValues[1]))
				distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
					otherFleet.InterX, otherFleet.InterY)
				if distance-gamestate.FleetSpeedPerSecond >= gamestate.OptimalFleetAttackRange {
					battle := galaxy.Protobuf.Battlemap[myFleet.FleetID]
					if battle == nil {
						galaxy.CreateOngoingBattle(&gamestate.Battle{})
					}
				} else if distance > gamestate.OptimalFleetAttackRange {
					myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
						myFleet.InterX, myFleet.InterY,
						otherFleet.InterY, otherFleet.InterY,
						gamestate.FleetSpeedPerSecond)
				}
			case gamestate.MoveFleetToAttackSystem:
				myFleet := galaxy.LookupFleetID(int32(action.ActionValues[0]))
				otherSystem := galaxy.LookupSystemName(action.ActionNames[0])
				distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
					otherSystem.InterX, otherSystem.InterY)
				if distance > gamestate.LightYear/100 {
					myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
						myFleet.InterX, myFleet.InterY,
						otherSystem.InterY, otherSystem.InterY,
						gamestate.FleetSpeedPerSecond)
				}
			case gamestate.MoveFleetToDefendSystem:
				myFleet := galaxy.LookupFleetID(int32(action.ActionValues[0]))
				mySystem := galaxy.LookupSystemName(action.ActionNames[0])
				distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
					mySystem.InterX, mySystem.InterY)
				if distance > gamestate.LightYear/100 {
					myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
						myFleet.InterX, myFleet.InterY,
						mySystem.InterY, mySystem.InterY,
						gamestate.FleetSpeedPerSecond)
				}
			case gamestate.MoveFleetToOngoingBattle:
				myFleet := galaxy.LookupFleetID(int32(action.ActionValues[0]))
				battle := galaxy.LookupBattleID(int32(action.ActionValues[1]))
				originFleet := galaxy.LookupFleetID(battle.GetBattleOriginFleetID())
				distance := DistanceBetweenPoints(myFleet.InterX, myFleet.InterY,
					originFleet.InterX, originFleet.InterY)
				if distance > gamestate.OptimalFleetAttackRange {
					myFleet.InterX, myFleet.InterY = vectors.MoveTowardsDestination(
						myFleet.InterX, myFleet.InterY,
						originFleet.InterY, originFleet.InterY,
						gamestate.FleetSpeedPerSecond)
				}
			}
		}
	})
	close(futureActionChan)
	wg.Wait()
}
