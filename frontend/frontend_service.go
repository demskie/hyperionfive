package frontend

import (
	"sync"
	"time"

	"github.com/demskie/hyperionfive/frontendmessage"
)

// Service ...
type Service struct {
	sync.RWMutex
	instanceName   string
	version        int32
	allServers     map[uint64]*frontendmessage.Backend
	addrToProfiles map[string]map[string]time.Time
}
