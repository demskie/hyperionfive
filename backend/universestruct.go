package backend

import (
	"hyperionfive/firewall"
	"hyperionfive/names"
	"hyperionfive/stringdatastore"
	"sync"
)

type UniverseStruct struct {
	sync.RWMutex
	GalaxyData           *GalaxyStruct
	GalaxyNameInt        int32
	GlobalX              int64
	GlobalY              int64
	ClientSignal         chan struct{}
	PlayerPtrs           map[int32][]*SolarSystem
	PlayerNameData       *stringdatastore.References
	SystemAndPlanetNames *names.NameGenerator
	PendingPlayerSpawns  []*playerInfo
	Firewall             *firewall.Checker
}

type playerInfo struct {
	uniqueID string
	strength int
}
