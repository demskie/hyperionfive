package backend

import (
	"fmt"

	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
)

// AdvanceProduction ...
func (galaxy *GalaxyStruct) AdvanceProduction() {
	const fName = "(advanceProduction)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	galaxy.ProcessingQueue.Execute(func(coreNum int) {
		startIndex, stopIndex := intutil.GetStartAndStopIndexForCore(
			coreNum, args.NumberOfCores, len(galaxy.AnyPlanetSlice))
		if stopIndex <= startIndex {
			return
		}
		rnum := galaxy.ProcessingRnumSlice[coreNum]
		startIndex += rnum.Intn(imprecisionFactor)
		var (
			requiredProgress int32
			planet           *gamestate.Planet
		)
		for planetIndex := startIndex; planetIndex < stopIndex; planetIndex += imprecisionFactor {
			planet = galaxy.AnyPlanetSlice[planetIndex]
			planet.Progress += int32(planet.Population/oneMillion) + 1
			switch planet.Focus {
			case gamestate.INCREASEPOPULATION:
				if planet.Population > oneMillion {
					requiredProgress = int32(int64(planet.Population/oneMillion) * int64(planet.Population/oneMillion))
				} else {
					requiredProgress = 100
				}
				if planet.Progress > requiredProgress {
					planet.Progress -= requiredProgress
					planet.Food++
				}
			case gamestate.BUILDTITAN:
				if planet.Progress > oneMillion {
					planet.Progress -= oneMillion

				}
			case gamestate.BUILDCAPITALSHIP:
				if planet.Progress > 10000 {
					planet.Progress -= 10000

				}
			case gamestate.BUILDFIGHTER:
				if planet.Progress > 100 {
					planet.Progress -= 100
				}
			default:
				log.Error(fName, fmt.Sprintf("Unknown obj.Focus value '%v'", planet.Focus))
			}
		}
	})
}
