package main

import "time"

type tickCounterData struct {
	lastTick                 time.Time
	overtimeCount            int
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
	prevLastTick := data.lastTick
	data.lastTick = time.Now()
	timeMultiplier := data.lastTick.Sub(prevLastTick).Seconds()
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
	if data.halfSecondCounter >= halfsecond {
		data.halfSecondCounter -= halfsecond
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
	const fName = "(tickCounterData)"
	switch {
	case data.twentyFiveSecondCounter > twentyfiveseconds:
		data.overtimeCount = 0
	case data.fiveSecondCounter > fiveseconds:
		data.overtimeCount = 0
	case data.oneSecondCounter > onesecond:
		data.overtimeCount = 0
	case data.halfSecondCounter > halfsecond:
		data.overtimeCount = 0
	case data.deciSecondCounter > decisecond:
		data.overtimeCount = 0
	case data.centiSecondCounter > centisecond:
		data.overtimeCount = 0
	default:
		data.overtimeCount++
		time.Sleep(time.Millisecond)
	}
	if data.overtimeCount%100 == 0 && data.overtimeCount != 0 {
		logError(fName, "overtime detected", text(data.overtimeCount), "times")
	}
	data.HasBeenFiveSeconds = false
	data.HasBeenOneSecond = false
	data.HasBeenOneSecond = false
	data.HasBeenOneHalfSecond = false
	data.HasBeenOneTenthSecond = false
	data.HasBeenOneHundrethSecond = false
}
