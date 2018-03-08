package main

import "time"

type tickCounterData struct {
	lastTick                 time.Time
	twentyFiveSecondCounter  float64
	HasBeenTwentyFiveSeconds bool
	fiveSecondCounter        float64
	HasBeenFiveSeconds       bool
	oneSecondCounter         float64
	HasBeenOneSecond         bool
	halfSecondCounter        float64
	HasBeenOneHalfSecond     bool
	deciSecondCounter        float64
	HasBeenOneTenthSecond    bool
	centiSecondCounter       float64
	HasBeenOneHundrethSecond bool
}

const (
	twentyfiveseconds = float64(25)
	fiveseconds       = float64(5)
	onesecond         = float64(1)
	halfsecond        = float64(1 / 2)
	decisecond        = float64(1 / 10)
	centisecond       = float64(1 / 100)
)

func (data *tickCounterData) Update() float64 {
	timeMultiplier := time.Since(data.lastTick).Seconds()
	data.lastTick = time.Now()
	backendCounters.IncrementTickCount()
	data.twentyFiveSecondCounter += timeMultiplier
	if data.twentyFiveSecondCounter >= twentyfiveseconds {
		data.twentyFiveSecondCounter -= twentyfiveseconds
		data.HasBeenTwentyFiveSeconds = true
	}
	data.fiveSecondCounter += timeMultiplier
	if data.fiveSecondCounter >= fiveseconds {
		data.fiveSecondCounter -= fiveseconds
		data.HasBeenFiveSeconds = true
	}
	data.oneSecondCounter += timeMultiplier
	if data.oneSecondCounter >= onesecond {
		data.oneSecondCounter -= onesecond
		data.HasBeenOneSecond = true
	}
	data.halfSecondCounter += timeMultiplier
	if data.halfSecondCounter >= onesecond {
		data.halfSecondCounter -= onesecond
		data.HasBeenOneHalfSecond = true
	}
	data.deciSecondCounter += timeMultiplier
	if data.deciSecondCounter >= decisecond {
		data.deciSecondCounter -= decisecond
		data.HasBeenOneTenthSecond = true
	}
	data.centiSecondCounter += timeMultiplier
	if data.centiSecondCounter >= centisecond {
		data.centiSecondCounter -= centisecond
		data.HasBeenOneHundrethSecond = true
	}
	return timeMultiplier
}

func (data *tickCounterData) WaitOneMillisecondAndReset() {
	time.Sleep(time.Millisecond)
	data.HasBeenFiveSeconds = false
	data.HasBeenOneSecond = false
	data.HasBeenOneSecond = false
	data.HasBeenOneHalfSecond = false
	data.HasBeenOneTenthSecond = false
	data.HasBeenOneHundrethSecond = false
}
