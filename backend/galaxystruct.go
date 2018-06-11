package backend

import (
	"math/rand"
	"sync"

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

	// all objects that have a population
	AnyPlanetMutex *sync.Mutex
	AnyPlanetSlice []*gamestate.Planet

	// neutral planets
	NeutralAnyPlanetMutex *sync.Mutex
	NeutralAnyPlanetSlice []*gamestate.Planet

	// used when spawning new players
	NeutralTerranPlanetMutex *sync.Mutex
	NeutralTerranPlanetSlice []*gamestate.Planet

	// used to get pointers based on ID or Name
	PointerMappings *PointerMappings

	// used when serializing data for players
	PlayerFogOfWarMutex *sync.Mutex
	PlayerFogOfWar      map[uint64]map[uint64]*gamestate.System // playerName // sysName // sysPtr

	// system and planet names
	SystemAndPlanetNames *names.NameGenerator

	// datastore ensuring no duplicate IDs
	FleetIDs  *intutil.Int32Generator
	BattleIDs *intutil.Int32Generator
	SiegeIDs  *intutil.Int32Generator
	ActionIDs *intutil.Int32Generator

	// used to lookup system by name
	SystemNameLookup map[uint64]*gamestate.System

	// used to get list of all fleets in system
	PlayerFleetInsideSystem map[uint64]map[uint64]*gamestate.Fleet // sysName // playerName // fleet
}
