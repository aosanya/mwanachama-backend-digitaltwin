package digitaltwin

// Closed vocabularies for the registry — asset types, asset lifecycle
// status, the station sub-kind, and directed-graph connection kinds. As
// with mwanachama-backend-accounting's vocabulary.go: none of these becomes
// a Postgres CHECK once a migration lands — a value added without an arm
// here should fail at the write, not relocate the same failure into a
// migration nobody re-reads.

// AssetType is the closed set of physical infrastructure types this twin
// tracks. Decided in the 2026-09-03 dev-research session (DSN-1701):
// pipelines and transmission lines were the driving examples, valves/
// stations/substations/sensors were named alongside them as "definitely"
// in scope.
type AssetType string

const (
	// AssetTypePipeline is a run of pipe carrying a fluid or gas.
	AssetTypePipeline AssetType = "pipeline"
	// AssetTypeTransmissionLine is an electrical transmission line.
	AssetTypeTransmissionLine AssetType = "transmission_line"
	// AssetTypeValve is an inline flow-control device on a Pipeline.
	AssetTypeValve AssetType = "valve"
	// AssetTypeStation is a compressor or pump station — see [StationType].
	AssetTypeStation AssetType = "station"
	// AssetTypeSubstation is an electrical substation on a TransmissionLine.
	AssetTypeSubstation AssetType = "substation"
	// AssetTypeSensor is a sensor or meter — the usual source of
	// [TelemetryReading]s, though a reading may also be posted directly
	// against any other asset type (DSN-1701 decision 6: the sensor is not
	// a mandatory intermediary).
	AssetTypeSensor AssetType = "sensor"
)

var assetTypes = map[AssetType]bool{
	AssetTypePipeline: true, AssetTypeTransmissionLine: true, AssetTypeValve: true,
	AssetTypeStation: true, AssetTypeSubstation: true, AssetTypeSensor: true,
}

// IsAssetType reports whether t is one of the six v1 asset types.
func IsAssetType(t AssetType) bool { return assetTypes[t] }

// StationType distinguishes a compressor station from a pump station. Only
// meaningful when Asset.AssetType == [AssetTypeStation]; empty (and
// rejected by Validate) for every other asset type.
type StationType string

const (
	StationTypeCompressor StationType = "compressor"
	StationTypePump       StationType = "pump"
)

var stationTypes = map[StationType]bool{StationTypeCompressor: true, StationTypePump: true}

// IsStationType reports whether t is one of the two station kinds.
func IsStationType(t StationType) bool { return stationTypes[t] }

// AssetStatus is the lifecycle state of an [Asset]. Hand-rolled state
// machine, the same shape taskmanager.TaskStatus.CanTransitionTo uses —
// pure Go, no storage dependency.
type AssetStatus string

const (
	// AssetStatusPlanned is the initial state — the asset is registered
	// but not yet in service.
	AssetStatusPlanned AssetStatus = "planned"
	// AssetStatusOperational means the asset is in service.
	AssetStatusOperational AssetStatus = "operational"
	// AssetStatusMaintenance means the asset is temporarily out of service
	// for planned work.
	AssetStatusMaintenance AssetStatus = "maintenance"
	// AssetStatusFault means the asset is out of service due to an
	// unplanned failure.
	AssetStatusFault AssetStatus = "fault"
	// AssetStatusDecommissioned is terminal — the asset has been
	// permanently retired.
	AssetStatusDecommissioned AssetStatus = "decommissioned"
)

var assetStatuses = map[AssetStatus]bool{
	AssetStatusPlanned: true, AssetStatusOperational: true, AssetStatusMaintenance: true,
	AssetStatusFault: true, AssetStatusDecommissioned: true,
}

// IsAssetStatus reports whether s is one of the five lifecycle states.
func IsAssetStatus(s AssetStatus) bool { return assetStatuses[s] }

// CanTransitionTo reports whether moving from the receiver status to next
// is a valid lifecycle step.
//
//	planned       → operational, decommissioned
//	operational   → maintenance, fault, decommissioned
//	maintenance   → operational, decommissioned
//	fault         → maintenance, operational, decommissioned
//	decommissioned → (none — terminal)
func (s AssetStatus) CanTransitionTo(next AssetStatus) bool {
	switch s {
	case AssetStatusPlanned:
		return next == AssetStatusOperational || next == AssetStatusDecommissioned
	case AssetStatusOperational:
		return next == AssetStatusMaintenance || next == AssetStatusFault || next == AssetStatusDecommissioned
	case AssetStatusMaintenance:
		return next == AssetStatusOperational || next == AssetStatusDecommissioned
	case AssetStatusFault:
		return next == AssetStatusMaintenance || next == AssetStatusOperational || next == AssetStatusDecommissioned
	default:
		return false // decommissioned is terminal
	}
}

// ConnectionKind is the directed-graph edge vocabulary between assets.
// DSN-1701 confirmed a directed graph was wanted but explicitly left the
// edge vocabulary open — these three are a working v1 set, not a closed
// decision; expect this list to grow once real topology is modeled.
type ConnectionKind string

const (
	// ConnectionConnectsTo is a generic directed link — the source asset
	// feeds into / is upstream of the target (e.g. Pipeline → Pipeline
	// segment, TransmissionLine → Substation).
	ConnectionConnectsTo ConnectionKind = "connects_to"
	// ConnectionPartOf marks the source asset as a physical component of
	// the target (e.g. Valve part_of Pipeline).
	ConnectionPartOf ConnectionKind = "part_of"
	// ConnectionMonitors links a Sensor to the asset it measures. Optional —
	// a reading may name an asset directly without a Sensor (DSN-1701
	// decision 6), so this edge documents instrumentation, not a
	// requirement for readings to exist.
	ConnectionMonitors ConnectionKind = "monitors"
)

var connectionKinds = map[ConnectionKind]bool{
	ConnectionConnectsTo: true, ConnectionPartOf: true, ConnectionMonitors: true,
}

// IsConnectionKind reports whether k is one of the current connection kinds.
func IsConnectionKind(k ConnectionKind) bool { return connectionKinds[k] }

// Direction selects edge orientation for [RegistryRepository.ListConnections].
type Direction int

const (
	// DirectionOutbound returns edges pointing AWAY from the start asset.
	DirectionOutbound Direction = iota
	// DirectionInbound returns edges pointing AT the start asset.
	DirectionInbound
)
