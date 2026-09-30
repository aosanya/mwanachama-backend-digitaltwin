package models

type Link struct {
	ID         string         `json:"id"`
	Relation   Relation       `json:"relation"`
	FromNodeID string         `json:"from_node_id"`
	ToNodeID   string         `json:"to_node_id"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

func (l Link) EndsApart() bool { return l.FromNodeID != l.ToNodeID }
