package digitaltwin

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func (m *twinManager) CreateNode(ctx context.Context, n Node) (Node, error) {
	if n.Status == "" {
		n.Status = models.StatusPlanned
	}
	if err := m.checks(roleNode, n); err != nil {
		return Node{}, err
	}
	n.ID = newID()
	n.CreatedAt = m.now()
	n.UpdatedAt = n.CreatedAt
	if err := m.insert(ctx, roleNode, n); err != nil {
		return Node{}, err
	}
	return n, nil
}

func (m *twinManager) GetNode(ctx context.Context, id string) (Node, error) {
	var out Node
	err := m.take(m.q(ctx, roleNode).Where("id = ?", id), roleNode, &out, ErrNodeNotFound)
	return out, err
}

func (m *twinManager) ListNodes(ctx context.Context, filter NodeFilter) ([]Node, error) {
	q := m.q(ctx, roleNode)
	if filter.Kind != "" {
		q = q.Where("kind = ?", filter.Kind)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", string(filter.Status))
	}
	return listOf[Node](m, q.Order("created_at").Limit(capped(filter.Limit)), roleNode)
}

func (m *twinManager) UpdateNode(ctx context.Context, n Node) (Node, error) {
	held, err := m.GetNode(ctx, n.ID)
	if err != nil {
		return Node{}, err
	}

	held.Name = n.Name
	held.Location = n.Location
	held.CommissionedAt = n.CommissionedAt
	held.Notes = n.Notes
	held.Doc = n.Doc
	held.UpdatedAt = m.now()

	if err := m.checks(roleNode, held); err != nil {
		return Node{}, err
	}
	if err := m.update(ctx, roleNode, "id", held.ID, held); err != nil {
		return Node{}, err
	}
	return held, nil
}

func (m *twinManager) SetNodeStatus(ctx context.Context, id string, next Status) (Node, error) {
	held, err := m.GetNode(ctx, id)
	if err != nil {
		return Node{}, err
	}
	if err := m.checksField(roleNode, "status", string(next)); err != nil {
		return Node{}, err
	}
	if !held.Status.CanTransitionTo(next) {
		return Node{}, fmt.Errorf("%w: a node that is %s cannot become %s", ErrInvalid, held.Status, next)
	}

	held.Status = next
	held.UpdatedAt = m.now()
	if err := m.update(ctx, roleNode, "id", held.ID, held); err != nil {
		return Node{}, err
	}
	return held, nil
}
