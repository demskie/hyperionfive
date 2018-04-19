package backend

import "time"

type tickCounterData struct {
	lastTick time.Time

	twentyFiveSecondCounter  time.Duration
	HasBeenTwentyFiveSeconds bool

	fiveSecondCounter  time.Duration
	HasBeenFiveSeconds bool

	oneSecondCounter time.Duration
	HasBeenOneSecond bool

	halfSecondCounter    time.Duration
	HasBeenOneHalfSecond bool

	deciSecondCounter     time.Duration
	HasBeenOneTenthSecond bool
}

const (
	oneminute         = time.Second * 60
	twentyfiveseconds = time.Second * 25
	fiveseconds       = time.Second * 5
	onesecond         = time.Second
	halfsecond        = time.Second / 2
	decisecond        = time.Second / 10
	waitTime          = time.Second / 1000
)

func (data *tickCounterData) Update() float64 {
	const fName = "(tickCounterData.Update)"
	timeSince := time.Since(data.lastTick)
	data.lastTick = data.lastTick.Add(timeSince)

	data.twentyFiveSecondCounter += timeSince
	data.fiveSecondCounter += timeSince
	data.oneSecondCounter += timeSince
	data.halfSecondCounter += timeSince
	data.deciSecondCounter += timeSince

	if data.twentyFiveSecondCounter >= twentyfiveseconds {
		data.twentyFiveSecondCounter = 0
		data.HasBeenTwentyFiveSeconds = true
	} else {
		data.HasBeenTwentyFiveSeconds = false
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

	if data.halfSecondCounter >= halfsecond {
		data.halfSecondCounter = 0
		data.HasBeenOneHalfSecond = true
	} else {
		data.HasBeenOneHalfSecond = false
	}

	if data.deciSecondCounter >= decisecond {
		data.deciSecondCounter = 0
		data.HasBeenOneTenthSecond = true
	} else {
		data.HasBeenOneTenthSecond = false
	}

	time.Sleep(waitTime)
	backendCounters.IncrementTickCount()
	return timeSince.Seconds()
}
