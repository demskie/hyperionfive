package backend

import (
	"fmt"
	"hyperionfive/intutil"
	"sync/atomic"
	"time"
)

var backendCounters *backendCounterData

func init() {
	backendCounters = &backendCounterData{}
}

type backendCounterData struct {
	numTicks         uint32
	numRequests      uint32
	numFighterSpawns uint32
}

func (data *backendCounterData) StartCounting() {
	const fName = "(backendProcessing)"
	go func() {
		for {
			time.Sleep(10 * time.Second)
			oldNumTicks := atomic.SwapUint32(&data.numTicks, 0)
			log.Telemetry(fName, fmt.Sprintf("less than %v ticks/sec", 1+(oldNumTicks/10)))
			// oldNumRequests := atomic.SwapUint32(&data.numRequests, 0)
			// logTelemetry(fName, "requestsPerSecond were", text(oldNumRequests/10))
			oldFightersPerSecond := atomic.SwapUint32(&data.numFighterSpawns, 0)
			log.Telemetry(fName, fmt.Sprintf("spawning %v fighters/sec",
				intutil.CommaFormatNum(oldFightersPerSecond/10)))
		}
	}()
}

func (data *backendCounterData) IncrementTickCount() {
	atomic.AddUint32(&data.numTicks, 1)
}

func (data *backendCounterData) IncrementRequestCount() {
	atomic.AddUint32(&data.numRequests, 1)
}

func (data *backendCounterData) IncrementFighterSpawnCount(val int32) {
	atomic.AddUint32(&data.numFighterSpawns, uint32(val))
}
