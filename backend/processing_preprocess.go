package backend

import "github.com/demskie/hyperionfive/backendmessage"

// PreprocessPopulationObjects ...
func (galaxy *GalaxyStruct) PreprocessPopulationObjects() {
	const fName = "(preprocessPopulationObjects)"
	log.Debug(fName, "starting")
	for _, system := range galaxy.Protobuf.System {
		for planetName := range system.ChildPlanet {
			planet := galaxy.Protobuf.Planet[planetName]
			galaxy.AnyPlanetSlice = append(galaxy.AnyPlanetSlice, planet)
			if planet.OwnerUint == 0 {
				galaxy.NeutralAnyPlanetSlice = append(galaxy.NeutralAnyPlanetSlice, planet)
			}
			if planet.OwnerUint == 0 && planet.PlanetType == backendmessage.TERRAN {
				galaxy.NeutralTerranPlanetSlice = append(galaxy.NeutralTerranPlanetSlice, planet)
			}
		}
	}
}
