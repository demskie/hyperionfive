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
	TITAN                       // can colonize, block point-defense and engage long-range
	CAPITALSHIP                 // long range ship-to-ship combat
	FIGHTER                     // close range combat
	FLAKCORVETTE                // block point-defense damage
	LARGETRADESHIP              // spawns and moves automatically from trading nations
	COLONIZER                   // colonize a single planet in new system
	_                           // _20_
)

// defining types of focus
const (
	INCREASEPOPULATION int32 = iota
	_
	BUILDTITAN
	BUILDCAPITALSHIP
	BUILDFIGHTER
	BUILDFLAKCORVETTE
	BUILDLARGETRADESHIP
	BUILDCOLONIZER
)

// GetObjectTypeString ...
func (obj *SolarObject) GetObjectTypeString() string {
	switch obj.ObjectType {
	case TERRAN:
		return "TERRAN"
	case DESERT:
		return "DESERT"
	case VOLCANIC:
		return "VOLCANIC"
	case ICE:
		return "ICE"
	case BARREN:
		return "BARREN"
	case FERROUS:
		return "FERROUS"
	case GREENHOUSE:
		return "GREENHOUSE"
	case OCEANIC:
		return "OCEANIC"
	case LARGEASTEROID:
		return "LARGEASTEROID"
	case DWARF:
		return "DWARF"
	case ASTEROIDBELT:
		return "ASTEROIDBELT"
	case PLASMASTORM:
		return "PLASMASTORM"
	case TITAN:
		return "TITAN"
	case CAPITALSHIP:
		return "CAPITALSHIP"
	case FIGHTER:
		return "FIGHTER"
	case FLAKCORVETTE:
		return "FLAKCORVETTE"
	case LARGETRADESHIP:
		return "LARGETRADESHIP"
	case COLONIZER:
		return "COLONIZER"
	}
	return "INVALID"
}

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

// CreateFleet ...
func (obj *SolarObject) CreateFleet() *GalacticFleet {
	if obj.IsPlanet() {
		switch obj.ObjectType {
		case TITAN:

		case CAPITALSHIP:
		case FIGHTER:
		case FLAKCORVETTE:
		case LARGETRADESHIP:
		case COLONIZER:
		}
		return &GalacticFleet{}
	}
	return &GalacticFleet{}
}
