package models

type Observation struct {
	ID         string  `json:"id"`
	NodeID     string  `json:"node_id"`
	NodeKind   string  `json:"node_kind,omitempty"`
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit,omitempty"`
	RecordedAt string  `json:"recorded_at"`
	IngestedAt string  `json:"ingested_at"`
	Quality    string  `json:"quality,omitempty"`
}

type ObservationFilter struct {
	NodeID     string `json:"node_id" query:"node_id"`
	MetricName string `json:"metric_name" query:"metric_name"`
	From       string `json:"from" query:"from"`
	To         string `json:"to" query:"to"`
	Limit      int    `json:"limit" query:"limit"`
}
