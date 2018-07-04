package backend

import "github.com/demskie/hyperionfive/gamestate"

// DeleteOngoingBattle ...
func (galaxy *GalaxyStruct) DeleteOngoingBattle(battle *gamestate.Battle) {
	galaxy.BattleMapMutex.Lock()
	galaxy.BattleIDs.Remove(battle.BattleID)
	//	delete(galaxy.PointerMappings.battleMappings, battle.BattleID)
	// BUG: must create deletion method
	rmv := battle.BattleID
	s := galaxy.Protobuf.Battleslice
	s[rmv] = s[len(s)-1]
	galaxy.Protobuf.Battleslice[len(s)-1] = nil
	galaxy.Protobuf.Battleslice = s[:len(s)-1]
	galaxy.BattleMapMutex.Unlock()
}
