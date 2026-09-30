package models

type Metric struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Unit        string   `json:"unit"`
	NodeKind    string   `json:"node_kind,omitempty"`
	Description string   `json:"description,omitempty"`
	MinValue    *float64 `json:"min_value,omitempty"`
	MaxValue    *float64 `json:"max_value,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func (m Metric) BoundsOrdered() bool {
	return m.MinValue == nil || m.MaxValue == nil || *m.MinValue <= *m.MaxValue
}

func (m Metric) Covers(nodeKind, name string) bool {
	if m.Name != name {
		return false
	}
	return m.NodeKind == "" || m.NodeKind == nodeKind
}
