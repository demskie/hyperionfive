package backend

import (
	"hyperionfive/names"
	"hyperionfive/simpletree"
	"math/rand"
	"sync"

	"github.com/AlexDemskie/simplesync"
)

// GalaxyStruct ...
type GalaxyStruct struct {
	// holds protobuf prestructured data
	Protobuf *Galaxy

	// created when generating galaxy to ensure objects are far enough apart
	Rtree *simpletree.Tree

	// secondary random number generator for use outside ProcessingQueue
	SingleRnumMutex *sync.Mutex
	SingleRnum      *rand.Rand

	// implement variable tick rate by multiplying against TimeMultiplier
	TickTime       *tickCounterData
	TimeMultiplier float64

	// worker queue implementation
	ProcessingQueue     *simplesync.WorkerPool
	ProcessingRnumSlice []*rand.Rand

	// track ai state of mind
	ActorStateSlice []*ActorState

	// all objects that have a population
	AnyPlanetMutex *sync.Mutex
	AnyPlanetSlice []*SolarObject

	// neutral planets
	NeutralAnyPlanetMutex *sync.Mutex
	NeutralAnyPlanetSlice []*SolarObject

	// used when spawning new players
	NeutralTerranPlanetMutex *sync.Mutex
	NeutralTerranPlanetSlice []*SolarObject

	// used to get pointers based on ID or Name
	PointerMappings *PointerMappings

	// used when serializing data for players
	PlayerFogOfWarMutex *sync.Mutex
	PlayerFogOfWar      map[int32]map[uint64]*SolarSystem // playerName // sysName // sysPtr

	// system and planet names
	SystemAndPlanetNames *names.NameGenerator
}
