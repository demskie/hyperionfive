package backendmessage

// GetBattleOriginFleetID ...
func (b *Battle) GetBattleOriginFleetID() int32 {
	for k := range b.AttackingFleetIDs {
		return k
	}
	return -1
}
