package backendmessage

import "math"

// global constants
const (
	GalaxyRadius                  = math.MaxInt32
	RealMilkyWayRadius            = 100000
	LightYear                     = GalaxyRadius / RealMilkyWayRadius
	MinimumDistanceBetweenSystems = 3 * LightYear
	MaxNumberOfPlayers            = 10000
	NumSystemsPerPlayer           = 25
	TotalNumSystems               = MaxNumberOfPlayers * NumSystemsPerPlayer
)
