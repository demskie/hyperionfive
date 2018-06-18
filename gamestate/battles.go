package gamestate

// GetBattleOriginFleetID ...
func (b *Battle) GetBattleOriginFleetID() int32 {
	return b.AttackingFleetIDs[0]
}
