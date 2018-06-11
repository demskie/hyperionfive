package gamestate

// Action types
const (
	MoveFleetToAttackFleet int32 = iota
	MoveFleetToAttackSystem
	MoveFleetToDefendSystem
	MoveFleetToOngoingBattle
	UndockFleetFromPlanet
)
