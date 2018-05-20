package gamestate

// solarsystem type
const (
	YELLOWGIANT int32 = iota // most desirable
	REDGIANT                 // mostly lava
	BLUEGIANT                // mostly ice
	NEUTRONSTAR              // just gas clouds
	WHITEDWARF               // just asteroids
	BLACKHOLE                // random, dangerous and AI spawn here
)
