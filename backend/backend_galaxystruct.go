package backend

import (
	"math/rand"

	"github.com/demskie/hyperionfive/backendmessage"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/spatialindex"
)

// GalaxyStruct ...
type GalaxyStruct struct {
	// holds protobuf prestructured data
	Protobuf *backendmessage.Galaxy

	// ensure systems are far enough apart
	Rtree *spatialindex.Tree

	// secondary random number generator for use outside ProcessingQueue
	SingleRnum *rand.Rand

	// implement variable tick rate by multiplying against TimeMultiplier
	TickTime       *tickCounterData
	TimeMultiplier float64

	// all objects that have a population
	AnyPlanetSlice []*backendmessage.Planet

	// neutral planets
	NeutralAnyPlanetSlice []*backendmessage.Planet

	// used when spawning new players
	NeutralTerranPlanetSlice []*backendmessage.Planet

	// datastore ensuring no duplicate IDs
	FleetIDs  *intutil.Int32Generator
	BattleIDs *intutil.Int32Generator
	SiegeIDs  *intutil.Int32Generator
	ActionIDs *intutil.Int32Generator
}
