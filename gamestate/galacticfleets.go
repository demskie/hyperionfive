package gamestate

// Power ...
func (fleet *GalacticFleet) Power() int64 {
	pressure := int64(fleet.TitanCount * 1000000)
	pressure += int64(fleet.CapitalCount * 10000)
	pressure += int64(fleet.FighterCount * 100)
	pressure += int64(fleet.FlakCorvetteCount * 100)
	pressure += int64(fleet.LargeTradeshipCount * 1)
	pressure += int64(fleet.ColonizerCount * 1)
	return pressure
}

// Merge ...
func (fleet *GalacticFleet) Merge(otherFleet *GalacticFleet) {

	return
}

// Split ...
func (fleet *GalacticFleet) Split(value int) (newFleet *GalacticFleet) {

	return
}

// Dock ...
func (fleet *GalacticFleet) Dock(obj *SolarObject) {

}
