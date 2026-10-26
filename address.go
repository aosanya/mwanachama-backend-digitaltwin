package digitaltwin

import (
	"context"

	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func (m *twinManager) MatchAddress(ctx context.Context, streetKind, street, number string) ([]Node, error) {
	want, ok := models.LeadingNumber(number)
	if !ok {
		return nil, nil
	}
	streets, err := m.MatchLabel(ctx, streetKind, street)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, s := range streets {
		links, err := m.ListLinks(ctx, s.ID, models.RelationPartOf, models.DirectionInbound)
		if err != nil {
			return nil, err
		}
		for _, l := range links {
			n, err := m.GetNode(ctx, l.FromNodeID)
			if err != nil {
				return nil, err
			}
			if got, ok := houseNumber(n); ok && got == want {
				out = append(out, n)
			}
		}
	}
	return out, nil
}

func houseNumber(n Node) (int, bool) {
	written, ok := n.Doc["house_number"].(string)
	if !ok {
		return 0, false
	}
	return models.LeadingNumber(written)
}
