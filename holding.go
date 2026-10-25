package digitaltwin

import (
	"context"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func (m *twinManager) DeepestHolding(ctx context.Context, kind string, at geo.Position) (Node, error) {
	q := m.q(ctx, roleNode).Where("shape IS NOT NULL")
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	candidates, err := listOf[Node](m, q, roleNode)
	if err != nil {
		return Node{}, err
	}
	var best Node
	bestSpan := -1.0
	for _, n := range candidates {
		if n.Shape == nil || !geo.Contains(*n.Shape, at) {
			continue
		}
		span := boxSpan(geo.BoundingBox(*n.Shape))
		if bestSpan < 0 || span < bestSpan {
			best, bestSpan = n, span
		}
	}
	if bestSpan < 0 {
		return Node{}, ErrNodeNotFound
	}
	return best, nil
}

func boxSpan(b geo.Box) float64 {
	return (b.MaxLon - b.MinLon) * (b.MaxLat - b.MinLat)
}
