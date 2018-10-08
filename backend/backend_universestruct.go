package backend

import (
	"bytes"
	"sync"

	"github.com/demskie/spatialindex"
)

// UniverseStruct ...
type UniverseStruct struct {
	sync.RWMutex
	GalaxyStruct        *GalaxyStruct
	UgridSystems        *spatialindex.Grid
	UgridFleets         *spatialindex.Grid
	UgridBattles        *spatialindex.Grid
	UgridSieges         *spatialindex.Grid
	UgridBytes          map[int]bytes.Buffer
	ClientSignal        chan struct{}
	PendingPlayerSpawns []*playerInfo
}

type playerInfo struct {
	nameUint uint64
	strength int
}
