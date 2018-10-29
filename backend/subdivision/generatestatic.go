package subdivision

import (
	"log"
	"math"
	"math/rand"

	"github.com/demskie/hyperionfive/backendmessage"
	"github.com/demskie/randutil"
	"github.com/demskie/spatialindex"
)

func generateStaticGalaxy(galaxyID, precision int32, globalX, globalY int64) *backendmessage.StaticGalaxy {
	rnum := randutil.CreateMathRnum(int64(galaxyID) * globalX * globalY)
	tree := spatialindex.NewTree()
	systems := map[int32]*backendmessage.StaticSystem{}
	for len(systems) < backendmessage.TotalNumSystems {
		ss := &backendmessage.StaticSystem{}
		var sysID int32
		for {
			sysID = rnum.Int31()
			_, exists := systems[sysID]
			if !exists {
				systems[sysID] = ss
				break
			}
		}
		for {
			generateEllipticalCoordinates(ss, rnum)
			x1, y1 := float64(ss.InterX), float64(ss.InterY)
			tree.Add(uint64(sysID), x1, y1)
			objects, err := tree.NearestNeighbors(uint64(sysID), 1)
			if err != spatialindex.ErrNotEnoughNeighbors && err != nil {
				log.Fatalln(err)
			}
			if len(systems) > 1 {
				x2, y2 := objects[0].X, objects[0].Y
				if math.Hypot(x1-x2, y1-y2) <= backendmessage.MinimumDistanceBetweenSystems {
					tree.Delete(uint64(sysID))
					continue
				}
			}
			ss.InterX = int32(x1)
			ss.InterY = int32(y1)
			break
		}
	}
	return &backendmessage.StaticGalaxy{
		Id:           galaxyID,
		GlobalX:      globalX,
		GlobalY:      globalY,
		Precision:    precision,
		StaticSystem: systems,
	}
}

func generateSpiralCoordinates(ss *backendmessage.StaticSystem, numArms int, rnum *rand.Rand) {
	// do something
}

func generateEllipticalCoordinates(ss *backendmessage.StaticSystem, rnum *rand.Rand) {
	var X, Y float64
	for {
		ss.InterX = rnum.Int31() * int32(randutil.GetPosOrNegOne(rnum))
		ss.InterY = rnum.Int31() * int32(randutil.GetPosOrNegOne(rnum))
		X = float64(ss.InterX * ss.InterX)
		Y = float64(ss.InterY * ss.InterY)
		if math.Sqrt(X+Y) >= backendmessage.GalaxyRadius {
			continue
		}
		break
	}
}
