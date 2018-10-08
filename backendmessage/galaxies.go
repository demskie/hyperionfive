package backendmessage

import "math"

// galaxy type
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)

// global constants
const (
	GalaxyRadius                  = math.MaxInt64
	RealMilkyWayRadius            = 100000
	LightYear                     = GalaxyRadius / RealMilkyWayRadius
	MinimumDistanceBetweenSystems = 3 * LightYear
)
