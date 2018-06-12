package backend

import (
	"github.com/demskie/hyperionbot/actor"
	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
)

// AdvanceActors ...
func (galaxy *GalaxyStruct) AdvanceActors() {
	const fName = "(advanceActors)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	allActions := make([][]*actor.ClientRequest, args.NumberOfCores)
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartStopForCore(
			coreNum, args.NumberOfCores, len(galaxy.Actors))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			bot           *actor.Bot
			partialGalaxy *gamestate.Galaxy
		)
		for botIndex := startIndex; botIndex < stopIndex; botIndex += imprecisionFactor {
			bot = galaxy.Actors[botIndex]
			partialGalaxy = &gamestate.Galaxy{
				NameUint:    galaxy.Protobuf.NameUint,
				GlobalX:     galaxy.Protobuf.GlobalX,
				GlobalY:     galaxy.Protobuf.GlobalY,
				Systemslice: galaxy.PlayerFogOfWarSlice(bot.GetOwner()),
			}
			allActions[coreNum] = bot.Process(partialGalaxy)
		}
	})
	for coreNum := range allActions {
		for _, action := range allActions[coreNum] {
			galaxy.CreateOutstandingAction(&gamestate.Action{
				ActionType:   action.ActionType,
				ActionValues: action.ActionValues,
				ActionNames:  action.ActionNames,
			})
		}
	}
}
