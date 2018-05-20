package backend

import "github.com/demskie/hyperionfive/gamestate"

// DeleteOngoingBattle ...
func (galaxy *GalaxyStruct) DeleteOngoingBattle(battle *gamestate.OngoingBattle) {
	galaxy.PointerMappings.ongoingBattleMutex.Lock()
	defer galaxy.PointerMappings.ongoingBattleMutex.Unlock()
	delete(galaxy.PointerMappings.ongoingBattleMappings, battle.BattleID)
	rmv := battle.BattleID
	s := galaxy.Protobuf.Ongoingbattleslice
	s[rmv] = s[len(s)-1]
	galaxy.Protobuf.Ongoingbattleslice[len(s)-1] = nil
	galaxy.Protobuf.Ongoingbattleslice = s[:len(s)-1]
}
