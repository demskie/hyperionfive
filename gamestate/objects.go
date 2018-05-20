package gamestate

// defining types of objects
const (
	TERRAN         int32 = iota // -20% population
	DESERT                      // -50% population, 2x logistics
	VOLCANIC                    // -75% population, lots of metal
	ICE                         // -50% population, lots of crystal
	BARREN                      // -90% population, 3x logistics
	FERROUS                     //
	GREENHOUSE                  //
	OCEANIC                     // max population
	LARGEASTEROID               //
	DWARF                       //
	_                           // _10_
	ASTEROIDBELT                //
	PLASMASTORM                 //
	_                           // _13_
	LARGETRADESHIP              // spawns and moves automatically from trading nations
	FIGHTER                     // close range combat
	FLAKCORVETTE                // block point-defense damage
	COLONIZER                   // colonize a single planet in new system
	CAPITALSHIP                 // long range ship-to-ship combat
	TITAN                       // can colonize, block point-defense and engage long-range
	_                           // _20_
)

// IsPlanet ...
func (obj *SolarObject) IsPlanet() bool {
	if obj.ObjectType < 10 {
		return true
	}
	return false
}

// IsShip ...
func (obj *SolarObject) IsShip() bool {
	if (13 < obj.ObjectType) && (obj.ObjectType < 20) {
		return true
	}
	return false
}
