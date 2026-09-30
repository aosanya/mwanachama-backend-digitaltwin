package digitaltwin

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-digitaltwin/models"
)

func (m *twinManager) CreateLink(ctx context.Context, l Link) (Link, error) {
	if err := m.checks(roleLink, l); err != nil {
		return Link{}, err
	}
	if !l.EndsApart() {
		return Link{}, fmt.Errorf("%w: a node cannot link to itself", ErrInvalid)
	}
	if err := m.bothEndsExist(ctx, l); err != nil {
		return Link{}, err
	}

	held, err := m.linkBetween(ctx, l.Relation, l.FromNodeID, l.ToNodeID)
	if err == nil {
		return held, nil
	}
	if !errors.Is(err, ErrLinkNotFound) {
		return Link{}, err
	}

	l.ID = newID()
	l.CreatedAt = m.now()
	if err := m.insert(ctx, roleLink, l); err != nil {
		return Link{}, err
	}
	return l, nil
}

func (m *twinManager) bothEndsExist(ctx context.Context, l Link) error {
	for _, id := range []string{l.FromNodeID, l.ToNodeID} {
		if _, err := m.GetNode(ctx, id); err != nil {
			if errors.Is(err, ErrNodeNotFound) {
				return fmt.Errorf("%w: no node %q", ErrInvalidLink, id)
			}
			return err
		}
	}
	return nil
}

func (m *twinManager) linkBetween(ctx context.Context, relation Relation, fromNodeID, toNodeID string) (Link, error) {
	q := m.q(ctx, roleLink).
		Where("relation = ?", string(relation)).
		Where("from_node_id = ?", fromNodeID).
		Where("to_node_id = ?", toNodeID)
	var out Link
	err := m.take(q, roleLink, &out, ErrLinkNotFound)
	return out, err
}

func (m *twinManager) DeleteLink(ctx context.Context, relation Relation, fromNodeID, toNodeID string) error {
	if _, err := m.linkBetween(ctx, relation, fromNodeID, toNodeID); err != nil {
		return err
	}
	return m.q(ctx, roleLink).
		Where("relation = ?", string(relation)).
		Where("from_node_id = ?", fromNodeID).
		Where("to_node_id = ?", toNodeID).
		Delete(nil).Error
}

func (m *twinManager) ListLinks(ctx context.Context, nodeID string, relation Relation, direction Direction) ([]Link, error) {
	column := "from_node_id"
	if direction == models.DirectionInbound {
		column = "to_node_id"
	}
	q := m.q(ctx, roleLink).Where(column+" = ?", nodeID)
	if relation != "" {
		q = q.Where("relation = ?", string(relation))
	}
	return listOf[Link](m, q.Order("created_at"), roleLink)
}
