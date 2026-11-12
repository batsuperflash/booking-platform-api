package storage

import (
	"context"
	"time"
)

type Slot struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resource_id"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateSlotParams struct {
	ResourceID string
	StartsAt   time.Time
	EndsAt     time.Time
}

func (p *Postgres) CreateSlot(ctx context.Context, params CreateSlotParams) (Slot, error) {
	query := `
		INSERT INTO slots (resource_id, starts_at, ends_at)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text, resource_id::text, starts_at, ends_at, created_at
	`

	var slot Slot

	err := p.Pool.QueryRow(
		ctx,
		query,
		params.ResourceID,
		params.StartsAt,
		params.EndsAt,
	).Scan(
		&slot.ID,
		&slot.ResourceID,
		&slot.StartsAt,
		&slot.EndsAt,
		&slot.CreatedAt,
	)

	if err != nil {
		return Slot{}, err
	}

	return slot, nil
}

func (p *Postgres) ListSlotsByResource(ctx context.Context, resourceID string) ([]Slot, error) {
	query := `
		SELECT id::text, resource_id::text, starts_at, ends_at, created_at
		FROM slots
		WHERE resource_id = $1::uuid
		ORDER BY starts_at ASC
	`
	rows, err := p.Pool.Query(ctx, query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]Slot, 0)

	for rows.Next() {
		var slot Slot

		err := rows.Scan(
			&slot.ID,
			&slot.ResourceID,
			&slot.StartsAt,
			&slot.EndsAt,
			&slot.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		slots = append(slots, slot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return slots, nil

}

func (p *Postgres) ListAvailableSlotsByResource(ctx context.Context, resourceID string) ([]Slot, error) {
	query := `
	    SELECT s.id::text, s.resource_id::text, s.starts_at, s.ends_at, s.created_at
		FROM slots s
		LEFT JOIN bookings b
			ON b.slot_id = s.id AND b.status = 'confirmed'
		WHERE s.resource_id = $1::uuid
			AND b.id IS NULL
		ORDER BY s.starts_at ASC
	`
	rows, err := p.Pool.Query(ctx, query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]Slot, 0)

	for rows.Next() {
		var slot Slot
		err := rows.Scan(
			&slot.ID,
			&slot.ResourceID,
			&slot.StartsAt,
			&slot.EndsAt,
			&slot.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		slots = append(slots, slot)

	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return slots, nil
}
