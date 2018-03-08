package main

import (
	"sync"
	"sync/atomic"

	"github.com/dhconnelly/rtreego"
)

// GatherAllPlayerPointers ...
func (gal *GalaxyData) GatherAllPlayerPointers() {
	const fName = "(gatherAllPlayerPointers)"
	mutex := sync.RWMutex{} // faster than you think
	numObjects := int64(0)
	gal.ProcessingQueue <- func(coreNum int) {
		startIndex, stopIndex := calculateCoreSlice(coreNum, len(gal.Data.Solarsystemslice))
		for ssIndex := startIndex; ssIndex < stopIndex; ssIndex++ {
			ss := gal.Data.Solarsystemslice[ssIndex]
			for _, obj := range ss.Solarobjectslice {
				atomic.AddInt64(&numObjects, 1)
				mutex.RLock()
				_, ownerExists := gal.PlayerPtrs[obj.Owner]
				if !ownerExists {
					mutex.RUnlock()
					mutex.Lock()
					gal.PlayerPtrs[obj.Owner] = []*SolarSystem{ss}
					mutex.Unlock()
				} else {
					foundMatch := false
					for _, anotherSS := range gal.PlayerPtrs[obj.Owner] {
						if ss == anotherSS {
							foundMatch = true
							break
						}
					}
					if foundMatch == false {
						mutex.RUnlock()
						mutex.Lock()
						gal.PlayerPtrs[obj.Owner] = append(
							gal.PlayerPtrs[obj.Owner], ss)
						mutex.Unlock()
					} else {
						mutex.RUnlock()
					}
				}
			}
		}
	}
	gal.ProcessingQueue <- func(coreNum int) {
		if coreNum == 0 {
			logInfo(fName, "processed", commaFormatNum(numObjects), "objects")
		}
	}
}

// GatherAllPlanetPointers ...
func (gal *GalaxyData) GatherAllPlanetPointers() {
	const fName = "(gatherAllPlanetPointers)"
	appendMutex := sync.Mutex{}
	tempSlice := make([][]*SolarObject, GLOBAL.numberOfCores)
	tempSliceTerran := make([][]*SolarObject, GLOBAL.numberOfCores)
	tempSliceNeutral := make([][]*SolarObject, GLOBAL.numberOfCores)
	for i := range tempSlice {
		tempSlice[i] = make([]*SolarObject, 0, oneMillion)
		tempSliceTerran[i] = make([]*SolarObject, 0, oneMillion)
		tempSliceNeutral[i] = make([]*SolarObject, 0, oneMillion)
	}
	gal.ProcessingQueue <- func(coreNum int) {
		startIndex, stopIndex := calculateCoreSlice(coreNum, len(gal.Data.Solarsystemslice))
		for ssIndex := startIndex; ssIndex < stopIndex; ssIndex++ {
			ss := gal.Data.Solarsystemslice[ssIndex]
			for _, ob := range ss.Solarobjectslice {
				if ob.IsPlanet() {
					tempSlice[coreNum] = append(tempSlice[coreNum], ob)
					if ob.Type == TERRAN {
						tempSliceTerran[coreNum] = append(tempSliceTerran[coreNum], ob)
					}
					if ob.Owner == "" {
						tempSliceNeutral[coreNum] = append(tempSliceNeutral[coreNum], ob)
					}
				}
			}
		}
		appendMutex.Lock()
		gal.PlanetPtrs = append(gal.PlanetPtrs, tempSlice[coreNum]...)
		gal.TerranPlanetPtrs = append(gal.TerranPlanetPtrs, tempSliceTerran[coreNum]...)
		gal.NeutralPlanetPtrs = append(gal.NeutralPlanetPtrs, tempSliceNeutral[coreNum]...)
		appendMutex.Unlock()
	}
}

// GenerateSystemLinks ...
func (galaxy *Galaxy) GenerateSystemLinks(rtree *rtreego.Rtree) map[string][]*SolarSystem {
	systemLinks := map[string][]*SolarSystem{}
	return systemLinks
}

// GatherAllObjectAndSystemNames ...
func (gal *GalaxyData) GatherAllObjectAndSystemNames() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for _, ss := range gal.Data.Solarsystemslice {
			for _, obj := range ss.Solarobjectslice {
				gal.ObjectNamePtrs[obj.Name] = obj
			}
		}
		wg.Done()
	}()
	go func() {
		for _, ss := range gal.Data.Solarsystemslice {
			gal.SystemNamePtrs[ss.Name] = ss
		}
		wg.Done()
	}()
	wg.Wait()
}

// only used by the following ChangeOwner() function
var globalChangeOwnerMutex sync.Mutex

// ChangeOwner ...
func (obj *SolarObject) ChangeOwner(newOwner string, gal *GalaxyData) {
	globalChangeOwnerMutex.Lock()
	defer globalChangeOwnerMutex.Unlock()
	originalOwner := obj.Owner
	if originalOwner == newOwner {
		return
	}
	ss := gal.SystemNamePtrs[obj.Name]
	addNew, deleteOriginal := true, false
	for _, otherObj := range ss.Solarobjectslice {
		if otherObj == obj {
			continue
		} else if addNew == false && deleteOriginal == true {
			break
		} else if otherObj.Owner == newOwner {
			addNew = false
		} else if otherObj.Owner == originalOwner {
			deleteOriginal = true
		}
	}
	if addNew {
		_, exists := GLOBAL.galaxyInfo.PlayerPtrs[newOwner]
		if !exists {
			GLOBAL.galaxyInfo.PlayerPtrs[newOwner] = []*SolarSystem{ss}
		} else {
			foundMatch := false
			for _, otherSS := range GLOBAL.galaxyInfo.PlayerPtrs[newOwner] {
				if ss == otherSS {
					foundMatch = true
					break
				}
			}
			if foundMatch == false {
				GLOBAL.galaxyInfo.PlayerPtrs[newOwner] = append(
					GLOBAL.galaxyInfo.PlayerPtrs[newOwner], ss)
			}
		}
	}
	if deleteOriginal {
		_, exists := GLOBAL.galaxyInfo.PlayerPtrs[originalOwner]
		if exists {
			newSlice := []*SolarSystem{}
			for _, otherSS := range GLOBAL.galaxyInfo.PlayerPtrs[originalOwner] {
				if ss != otherSS {
					newSlice = append(newSlice, ss)
				}
			}
			GLOBAL.galaxyInfo.PlayerPtrs[originalOwner] = newSlice
		}
	}
	obj.Owner = newOwner
}

// IsPlanet ...
func (obj *SolarObject) IsPlanet() bool {
	if obj.Type < 10 {
		return true
	}
	return false
}

// IsShip ...
func (obj *SolarObject) IsShip() bool {
	if (13 < obj.Type) && (obj.Type < 20) {
		return true
	}
	return false
}

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

// solarsystem type
const (
	YELLOWGIANT int32 = iota // most desirable
	REDGIANT                 // mostly lava
	BLUEGIANT                // mostly ice
	NEUTRONSTAR              // just gas clouds
	WHITEDWARF               // just asteroids
	BLACKHOLE                // random, dangerous and AI spawn here
)

// galaxy type
const (
	ELLIPTICAL int32 = iota
	SPIRAL
	BARREDSPIRAL
	IRREGULAR
)
