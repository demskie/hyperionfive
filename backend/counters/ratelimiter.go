package counters

import "time"

type RateLimiter struct {
	lastTick             time.Time
	oneMinuteCounter     time.Duration
	HasBeenOneMinute     bool
	thirtySecondCounter  time.Duration
	HasBeenThirtySeconds bool
	fiveSecondCounter    time.Duration
	HasBeenFiveSeconds   bool
	oneSecondCounter     time.Duration
	HasBeenOneSecond     bool
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lastTick: time.Now(),
	}
}

const (
	oneminute      = time.Second * 60
	thirtyseconds  = time.Second * 30
	fiveseconds    = time.Second * 5
	onesecond      = time.Second
	onemillisecond = time.Second / 1000
)

func (r *RateLimiter) Update() float64 {
	timeSince := time.Since(r.lastTick)
	r.lastTick = r.lastTick.Add(timeSince)
	r.oneMinuteCounter += timeSince
	r.thirtySecondCounter += timeSince
	r.fiveSecondCounter += timeSince
	r.oneSecondCounter += timeSince
	if r.thirtySecondCounter >= thirtyseconds {
		r.thirtySecondCounter = 0
		r.HasBeenThirtySeconds = true
	} else {
		r.HasBeenThirtySeconds = false
	}
	if r.fiveSecondCounter >= fiveseconds {
		r.fiveSecondCounter = 0
		r.HasBeenFiveSeconds = true
	} else {
		r.HasBeenFiveSeconds = false
	}
	if r.oneSecondCounter >= onesecond {
		r.oneSecondCounter = 0
		r.HasBeenOneSecond = true
	} else {
		r.HasBeenOneSecond = false
	}
	time.Sleep(onemillisecond)
	return timeSince.Seconds()
}
