package counters

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/demskie/logging"
)

type PerfStats struct {
	log              *logging.Logger
	numTicks         uint32
	numRequests      uint32
	numFighterSpawns uint32
}

func NewPerfStats(logger *logging.Logger) *PerfStats {
	p := &PerfStats{
		log: logger,
	}
	const fName = "(backendProcessing)"
	go func() {
		for {
			time.Sleep(10 * time.Second)
			oldNumTicks := atomic.SwapUint32(&p.numTicks, 0)
			p.log.Telemetry(fName, fmt.Sprintf("less than %v ticks/sec", 1+(oldNumTicks/10)))
			oldNumRequests := atomic.SwapUint32(&p.numRequests, 0)
			p.log.Telemetry(fName, fmt.Sprintf("%v requests/sec", oldNumRequests/10))
		}
	}()
	return p
}

func (p *PerfStats) IncrementTickCount() {
	atomic.AddUint32(&p.numTicks, 1)
}

func (p *PerfStats) IncrementRequestCount() {
	atomic.AddUint32(&p.numRequests, 1)
}
