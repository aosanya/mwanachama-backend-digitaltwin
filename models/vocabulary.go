package models

type Status string

const (
	StatusPlanned        Status = "planned"
	StatusOperational    Status = "operational"
	StatusMaintenance    Status = "maintenance"
	StatusFault          Status = "fault"
	StatusDecommissioned Status = "decommissioned"
)

func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case StatusPlanned:
		return next == StatusOperational || next == StatusDecommissioned
	case StatusOperational:
		return next == StatusMaintenance || next == StatusFault || next == StatusDecommissioned
	case StatusMaintenance:
		return next == StatusOperational || next == StatusDecommissioned
	case StatusFault:
		return next == StatusMaintenance || next == StatusOperational || next == StatusDecommissioned
	default:
		return false
	}
}

type Relation string

const (
	RelationConnectsTo Relation = "connects_to"
	RelationPartOf     Relation = "part_of"
	RelationMonitors   Relation = "monitors"
)

type Direction string

const (
	DirectionOutbound Direction = "outbound"
	DirectionInbound  Direction = "inbound"
)
