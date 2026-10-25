package digitaltwin

import (
	"context"

	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func (m *twinManager) MatchLabel(ctx context.Context, kind, text string) ([]Node, error) {
	want := models.NormalLabel(text)
	if want == "" {
		return nil, nil
	}
	q := m.q(ctx, roleNode)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	candidates, err := listOf[Node](m, q.Order("created_at"), roleNode)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, n := range candidates {
		if knownAs(n, want) {
			out = append(out, n)
		}
	}
	return out, nil
}

func knownAs(n Node, want string) bool {
	if models.NormalLabel(n.Name) == want {
		return true
	}
	for _, l := range n.Labels {
		if models.NormalLabel(l.Text) == want {
			return true
		}
	}
	return false
}
