package gamestate

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
