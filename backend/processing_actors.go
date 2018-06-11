package backend

import (
	"github.com/demskie/hyperionbot/actor"
	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
)

// AdvanceActors ...
func (galaxy *GalaxyStruct) AdvanceActors(universe *UniverseStruct) {
	const fName = "(advanceActors)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	allActions := make([][]*actor.ClientRequest, args.NumberOfCores)
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(universe.Actors))
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
			bot = universe.Actors[botIndex]
			partialGalaxy = &gamestate.Galaxy{
				NameUint:    universe.GalaxyNameUint,
				GlobalX:     universe.GlobalX,
				GlobalY:     universe.GlobalY,
				Systemslice: universe.PlayerPtrs[bot.GetOwner()],
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
