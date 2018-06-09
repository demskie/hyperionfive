package backend

import "time"

type tickCounterData struct {
	lastTick time.Time

	oneMinuteCounter time.Duration
	HasBeenOneMinute bool

	thirtySecondCounter  time.Duration
	HasBeenThirtySeconds bool

	fiveSecondCounter  time.Duration
	HasBeenFiveSeconds bool

	oneSecondCounter time.Duration
	HasBeenOneSecond bool
}

const (
	oneminute     = time.Second * 60
	thirtyseconds = time.Second * 30
	fiveseconds   = time.Second * 5
	onesecond     = time.Second
	waitTime      = time.Second / 1000
)

func (data *tickCounterData) Update() float64 {
	const fName = "(tickCounterData.Update)"
	timeSince := time.Since(data.lastTick)
	data.lastTick = data.lastTick.Add(timeSince)

	data.oneMinuteCounter += timeSince
	data.thirtySecondCounter += timeSince
	data.fiveSecondCounter += timeSince
	data.oneSecondCounter += timeSince

	if data.thirtySecondCounter >= thirtyseconds {
		data.thirtySecondCounter = 0
		data.HasBeenThirtySeconds = true
	} else {
		data.HasBeenThirtySeconds = false
	}

	if data.fiveSecondCounter >= fiveseconds {
		data.fiveSecondCounter = 0
		data.HasBeenFiveSeconds = true
	} else {
		data.HasBeenFiveSeconds = false
	}

	if data.oneSecondCounter >= onesecond {
		data.oneSecondCounter = 0
		data.HasBeenOneSecond = true
	} else {
		data.HasBeenOneSecond = false
	}

	time.Sleep(waitTime)
	backendCounters.IncrementTickCount()
	return timeSince.Seconds()
}
