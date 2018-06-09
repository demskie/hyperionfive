package gamestate

// OustandingAction types
const (
	MoveFleetToAttackFleet int32 = iota
	MoveFleetToAttackSystem
	MoveFleetToDefendSystem
	MoveFleetToOngoingBattle
	UndockFleetFromPlanet
)
