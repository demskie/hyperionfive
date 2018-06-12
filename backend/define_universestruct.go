package backend

import (
	"sync"

	"github.com/demskie/hyperionfive/names"
	"github.com/demskie/hyperionfive/stringdatastore"
)

// UniverseStruct ...
type UniverseStruct struct {
	sync.RWMutex
	GalaxyStruct         *GalaxyStruct
	ClientSignal         chan struct{}
	PlayerNames          *stringdatastore.References
	SystemAndPlanetNames *names.NameGenerator
	PendingPlayerSpawns  []*playerInfo
}

type playerInfo struct {
	nameUint uint64
	strength int
}
