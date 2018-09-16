package backend

import (
	"math/rand"
	"sync"

	"github.com/demskie/hyperionbot/actor"
	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/intutil"
	"github.com/demskie/hyperionfive/names"
	"github.com/demskie/spatialindex"

	"github.com/demskie/simplesync"
)

// GalaxyStruct ...
type GalaxyStruct struct {
	// holds protobuf prestructured data
	Protobuf *gamestate.Galaxy

	// locks for protobuf maps
	ObjectMapMutex *sync.RWMutex
	FleetMapMutex  *sync.RWMutex
	BattleMapMutex *sync.RWMutex
	SiegeMapMutex  *sync.RWMutex
	ActionMapMutex *sync.RWMutex

	// created when generating galaxy to ensure objects are far enough apart
	Rtree *spatialindex.Tree

	// secondary random number generator for use outside ProcessingQueue
	SingleRnumMutex *sync.Mutex
	SingleRnum      *rand.Rand

	// implement variable tick rate by multiplying against TimeMultiplier
	TickTime       *tickCounterData
	TimeMultiplier float64

	// worker queue implementation
	ProcessingQueue     *simplesync.WorkerPool
	ProcessingRnumSlice []*rand.Rand

	// System slice for parallel execution
	SystemSlice []*gamestate.System
	SystemNames []uint64

	// each player has a bot that micromanages units
	Actors []*actor.Bot

	// all objects that have a population
	AnyPlanetMutex *sync.Mutex
	AnyPlanetSlice []*gamestate.Planet

	// neutral planets
	NeutralAnyPlanetMutex *sync.Mutex
	NeutralAnyPlanetSlice []*gamestate.Planet

	// used when spawning new players
	NeutralTerranPlanetMutex *sync.Mutex
	NeutralTerranPlanetSlice []*gamestate.Planet

	// system and planet names
	SystemAndPlanetNames *names.NameGenerator

	// datastore ensuring no duplicate IDs
	FleetIDs  *intutil.Int32Generator
	BattleIDs *intutil.Int32Generator
	SiegeIDs  *intutil.Int32Generator
	ActionIDs *intutil.Int32Generator

	// used to get list of all fleets in system
	PlayerFleetInsideSystem map[uint64]map[uint64]*gamestate.Fleet // sysName // playerName // fleet
}
