package main

func makeGalaxyBuffer(originalGalaxy *Galaxy) (galBuf *Galaxy) {
	galBuf = &Galaxy{
		Name:             originalGalaxy.Name,
		GlobalX:          originalGalaxy.GlobalX,
		GlobalY:          originalGalaxy.GlobalY,
		Type:             originalGalaxy.Type,
		Solarsystemslice: make([]*SolarSystem, len(originalGalaxy.Solarsystemslice)),
	}
	for sysIndex, originalSystem := range originalGalaxy.Solarsystemslice {
		galBuf.Solarsystemslice[sysIndex] = &SolarSystem{
			Name:             originalSystem.Name,
			InterX:           originalSystem.InterX,
			InterY:           originalSystem.InterY,
			Type:             originalSystem.Type,
			Solarobjectslice: make([]*SolarObject, len(originalSystem.Solarobjectslice)),
		}
		for objIndex, originalObject := range originalSystem.Solarobjectslice {
			galBuf.Solarsystemslice[sysIndex].Solarobjectslice[objIndex] = &SolarObject{
				Name:       originalObject.Name,
				LocalX:     originalObject.LocalX,
				LocalY:     originalObject.LocalY,
				Owner:      originalObject.Owner,
				Food:       originalObject.Food,
				Population: originalObject.Population,
				Type:       originalObject.Type,
				Health:     originalObject.Health,
			}
		}
	}
	return galBuf
}

func deepCopyGalaxy(originalGalaxy, bufferGalaxy *Galaxy, indexMap map[string]int) {
	bufferGalaxy.Name = originalGalaxy.Name
	bufferGalaxy.GlobalX = originalGalaxy.GlobalX
	bufferGalaxy.GlobalY = originalGalaxy.GlobalY
	bufferGalaxy.Type = originalGalaxy.Type
	parallelProcessSystems(originalGalaxy, func(originalSystem *SolarSystem, coreID int) {
		bufferSystem := bufferGalaxy.Solarsystemslice[indexMap[originalSystem.Name]]
		bufferSystem.Name = originalSystem.Name
		bufferSystem.InterX = originalSystem.InterX
		bufferSystem.InterY = originalSystem.InterY
		bufferSystem.Type = originalSystem.Type
		if len(bufferSystem.Solarobjectslice) != len(originalSystem.Solarobjectslice) {
			if len(bufferSystem.Solarobjectslice) > len(originalSystem.Solarobjectslice) {
				bufferSystem.Solarobjectslice = append(bufferSystem.Solarobjectslice,
					bufferSystem.Solarobjectslice[:len(originalSystem.Solarobjectslice)]...)
			} else if len(bufferSystem.Solarobjectslice) < len(originalSystem.Solarobjectslice) {
				bufferSystem.Solarobjectslice = append(bufferSystem.Solarobjectslice,
					originalSystem.Solarobjectslice[len(bufferSystem.Solarobjectslice):]...)
			}
		}
		for idx, originalObject := range originalSystem.Solarobjectslice {
			bufferObject := bufferSystem.Solarobjectslice[idx]
			bufferObject.Name = originalObject.Name
			bufferObject.LocalX = originalObject.LocalX
			bufferObject.LocalY = originalObject.LocalY
			bufferObject.Owner = originalObject.Owner
			bufferObject.Food = originalObject.Food
			bufferObject.Population = originalObject.Population
			bufferObject.Type = originalObject.Type
			bufferObject.Health = originalObject.Health
		}
	})
}
