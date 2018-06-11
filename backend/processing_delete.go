package backend

import "github.com/demskie/hyperionfive/gamestate"

// DeleteOngoingBattle ...
func (galaxy *GalaxyStruct) DeleteOngoingBattle(battle *gamestate.Battle) {
	galaxy.PointerMappings.battleMutex.Lock()
	galaxy.BattleIDs.Remove(battle.BattleID)
	delete(galaxy.PointerMappings.battleMappings, battle.BattleID)
	rmv := battle.BattleID
	s := galaxy.Protobuf.Battleslice
	s[rmv] = s[len(s)-1]
	galaxy.Protobuf.Battleslice[len(s)-1] = nil
	galaxy.Protobuf.Battleslice = s[:len(s)-1]
	galaxy.PointerMappings.battleMutex.Unlock()
}
