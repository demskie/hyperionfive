package backend

import (
	"fmt"
	"math"

	"github.com/demskie/hyperionfive/backendmessage"
)

// AdvanceProduction ...
func (galaxy *GalaxyStruct) AdvanceProduction() {
	const fName = "(advanceProduction)"
	const imprecisionFactor = 10
	if galaxy.TickTime.HasBeenOneSecond == false {
		return
	}
	const (
		progressExponent = float64(2)
	)
	var (
		millionsPlusOne  int32
		requiredProgress int32
		planet           *backendmessage.Planet
		startIndex       = galaxy.SingleRnum.Intn(imprecisionFactor)
	)
	for planetIndex := startIndex; planetIndex < len(galaxy.AnyPlanetSlice); planetIndex += imprecisionFactor {
		planet = galaxy.AnyPlanetSlice[planetIndex]
		millionsPlusOne = int32(planet.Population/oneMillion) + 1
		planet.Progress += millionsPlusOne
		planet.DockedFighterCount += galaxy.SingleRnum.Int31n(millionsPlusOne)
		requiredProgress = getRequiredProgress(planet.Food)
		if args.DebugModeBool && planetIndex == 0 {
			log.Debug(fmt.Sprintf("current: %v required: %v", planet.Progress, requiredProgress))
		}
		if planet.Progress >= requiredProgress {
			planet.Progress -= requiredProgress
			planet.Food++
			if args.DebugModeBool && planetIndex == 0 {
				log.Debug(fmt.Sprintf("food increased to: %v", planet.Food))
			}
		}
		if planet.Progress > 100 {
			planet.Progress -= 100
			planet.DockedFighterCount++
		}
	}
}

func getRequiredProgress(food int32) int32 {
	switch food {
	case 0:
		return 4
	case 1:
		return 4
	case 2:
		return 8
	case 3:
		return 8
	case 4:
		return 8
	case 5:
		return 16
	case 6:
		return 16
	case 7:
		return 32
	case 8:
		return 32
	case 9:
		return 64
	case 10:
		return 128
	case 11:
		return 256
	case 12:
		return 512
	case 13:
		return 1024
	case 14:
		return 2048
	case 15:
		return 4096
	case 16:
		return 4096
	case 17:
		return 4096
	case 18:
		return 4096
	case 19:
		return 8192
	case 20:
		return 8192
	case 21:
		return 16384
	case 22:
		return 16384
	case 23:
		return 32768
	case 24:
		return 65536
	case 25:
		return 131072
	case 26:
		return 262144
	case 27:
		return 524288
	case 28:
		return 1048576
	case 29:
		return 2097152
	case 30:
		return 4194304
	case 31:
		return 8388608
	case 32:
		return 2147483647
	}
	return math.MaxInt32
}
