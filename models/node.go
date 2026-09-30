package models

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Node struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Kind           string         `json:"kind"`
	Status         Status         `json:"status"`
	Location       *Location      `json:"location,omitempty"`
	CommissionedAt string         `json:"commissioned_at,omitempty"`
	Notes          string         `json:"notes,omitempty"`
	Doc            map[string]any `json:"doc,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

type NodeFilter struct {
	Kind   string `json:"kind" query:"kind"`
	Status Status `json:"status" query:"status"`
	Limit  int    `json:"limit" query:"limit"`
}
