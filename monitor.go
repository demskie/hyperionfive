package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type atomicCounters struct {
	processGalaxyTime int64
	numTicks          uint32
	numRequests       uint32
}

func (c atomicCounters) Start() {
	const fName = "(monitorCounters)"
	go func() {
		for {
			time.Sleep(1 * time.Second)
			oldNumTicks := atomic.SwapUint32(&c.numTicks, 0)
			logTelemetry(fName, "ticksPerSecond:", fmt.Sprint(oldNumTicks))
		}
	}()
	go func() {
		for {
			time.Sleep(5 * time.Second)
			oldNumRequests := atomic.SwapUint32(&c.numRequests, 0)
			logTelemetry(fName, "requestsPerSecond:", fmt.Sprint(oldNumRequests/5))
		}
	}()
}

func (c atomicCounters) IncrementTickCount() {
	atomic.AddUint32(&c.numTicks, 1)
}

func (c atomicCounters) IncrementRequestCount() {
	atomic.AddUint32(&c.numRequests, 1)
}
