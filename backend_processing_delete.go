package main

// DeleteOngoingBattle ...
func (galaxy *GalaxyStruct) DeleteOngoingBattle(battle *OngoingBattle) {
	rmv := battle.BattleID
	s := galaxy.Protobuf.Ongoingbattleslice
	s[rmv] = s[len(s)-1]
	galaxy.Protobuf.Ongoingbattleslice[len(s)-1] = nil
	galaxy.Protobuf.Ongoingbattleslice = s[:len(s)-1]
}
