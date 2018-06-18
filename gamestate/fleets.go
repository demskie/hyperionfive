package gamestate

import "math"

// Power ...
func (fleet *Fleet) Power() int64 {
	pressure := int64(fleet.TitanCount * 1000000)
	pressure += int64(fleet.CapitalCount * 10000)
	pressure += int64(fleet.FighterCount * 100)
	return pressure
}

// Merge ...
func (fleet *Fleet) Merge(otherFleet *Fleet) {

	return
}

// Split ...
func (fleet *Fleet) Split(value int) (newFleet *Fleet) {

	return
}

// DistanceFromFleet ...
func (fleet *Fleet) DistanceFromFleet(otherFleet *Fleet) uint64 {
	x1, y1 := float64(fleet.InterX), float64(fleet.InterY)
	x2, y2 := float64(otherFleet.InterX), float64(otherFleet.InterY)
	return uint64(math.Hypot(float64(x1)-float64(x2), float64(y1)-float64(y2)))
}
