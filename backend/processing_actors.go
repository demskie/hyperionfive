package backend

import "hyperionfive/intutil"

// ProcessActors ...
func (galaxy *GalaxyStruct) ProcessActors() {
	const fName = "(processActors)"
	const imprecisionFactor = 10 * 10
	if galaxy.TickTime.HasBeenOneTenthSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(galaxy.ActorStateSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			actor *ActorState
		)
		for i := startIndex; i < stopIndex; i += imprecisionFactor {
			actor = galaxy.ActorStateSlice[i]
			actor.Id++
		}
	})
}

// ~/OneDrive/GoPrototypes/bin/protoc --go_out=/Users/thedemskie/OneDrive/GoPrototypes/src/hyperionfive/ --proto_path=/Users/thedemskie/OneDrive/GoPrototypes/src/hyperionfive/ protobuf_actor.proto
