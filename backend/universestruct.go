package backend

import (
	"sync"

	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/hyperionfive/gamestate"
	"github.com/demskie/hyperionfive/names"
	"github.com/demskie/hyperionfive/stringdatastore"
)

// UniverseStruct ...
type UniverseStruct struct {
	sync.RWMutex
	GalaxyData           *GalaxyStruct
	GalaxyNameUint       uint64
	GlobalX              int64
	GlobalY              int64
	ClientSignal         chan struct{}
	PlayerPtrs           map[int32][]*gamestate.SolarSystem
	PlayerNameData       *stringdatastore.References
	SystemAndPlanetNames *names.NameGenerator
	PendingPlayerSpawns  []*playerInfo
	Firewall             *firewall.Checker
}

type playerInfo struct {
	uniqueID string
	strength int
}
