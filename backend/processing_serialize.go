package backend

import "github.com/demskie/hyperionfive/backendmessage"

// SerializeSystems ...
func (universe *UniverseStruct) SerializeSystems() {
	const fName = "(serializeSystems)"
	for i, protoBuffer := range universe.UgridBytes {
		partialGalaxy := &backendmessage.Galaxy{
			NameUint:   universe.GalaxyStruct.Protobuf.NameUint,
			GlobalX:    universe.GalaxyStruct.Protobuf.GlobalX,
			GlobalY:    universe.GalaxyStruct.Protobuf.GlobalY,
			GalaxyType: universe.GalaxyStruct.Protobuf.GalaxyType,
		}
		points := universe.UgridSystems.CopyUnderlyingBucketValues(i)
		for _, point := range points {
			partialGalaxy.System[point.ID] = universe.GalaxyStruct.Protobuf.System[point.ID]
			for planetID := range partialGalaxy.System[point.ID].ChildPlanet {
				partialGalaxy.Planet[planetID] = universe.GalaxyStruct.Protobuf.Planet[planetID]
			}
			for objectID := range partialGalaxy.System[point.ID].ChildObject {
				partialGalaxy.Object[objectID] = universe.GalaxyStruct.Protobuf.Object[objectID]
			}
		}
		// TODO: need to include fleets, battles and sieges
		log.CheckError(protoBuffer.Marshal(partialGalaxy), fName)
	}
}
