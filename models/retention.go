package models

type Retention struct {
	ID            string `json:"id"`
	NodeKind      string `json:"node_kind,omitempty"`
	MetricName    string `json:"metric_name,omitempty"`
	RetentionDays int    `json:"retention_days"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func (r Retention) KeepsForAPositivePeriod() bool { return r.RetentionDays > 0 }

func (r Retention) ScopeIsOrdered() bool { return r.MetricName == "" || r.NodeKind != "" }

func (r Retention) Covers(nodeKind, metricName string) bool {
	if r.NodeKind != "" && r.NodeKind != nodeKind {
		return false
	}
	return r.MetricName == "" || r.MetricName == metricName
}
