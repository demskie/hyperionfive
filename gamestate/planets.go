package gamestate

// defining types of planets
const (
	TERRAN        int32 = iota // -20% population
	DESERT                     // -50% population, 2x logistics
	VOLCANIC                   // -75% population, lots of metal
	ICE                        // -50% population, lots of crystal
	BARREN                     // -90% population, 3x logistics
	FERROUS                    //
	GREENHOUSE                 //
	OCEANIC                    // max population
	LARGEASTEROID              //
	DWARF                      //
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

// GetPlanetTypeString ...
func (pl *Planet) GetPlanetTypeString() string {
	switch pl.PlanetType {
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
	}
	return "INVALID"
}
