package backend

import (
	"sync"

	"github.com/demskie/spatialindex"
	"github.com/gogo/protobuf/proto"
)

// UniverseStruct ...
type UniverseStruct struct {
	sync.RWMutex
	GalaxyStruct        *GalaxyStruct
	UgridSystems        *spatialindex.Grid
	UgridFleets         *spatialindex.Grid
	UgridBattles        *spatialindex.Grid
	UgridSieges         *spatialindex.Grid
	UgridBytes          [65535]proto.Buffer
	ClientSignal        chan struct{}
	PendingPlayerSpawns []*playerInfo
}

type playerInfo struct {
	nameUint uint64
	strength int
}
