package storage

import (
	"context"
	"time"
)

type Resource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateResourceParams struct {
	Name        string
	Type        string
	Description string
}

func (p *Postgres) CreateResource(ctx context.Context, params CreateResourceParams) (Resource, error) {
	query := `
		INSERT INTO resources (name, type, description)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, type, COALESCE(description, ''), created_at
	`

	var resource Resource

	err := p.Pool.QueryRow(
		ctx,
		query,
		params.Name,
		params.Type,
		params.Description,
	).Scan(
		&resource.ID,
		&resource.Name,
		&resource.Type,
		&resource.Description,
		&resource.CreatedAt,
	)

	if err != nil {
		return Resource{}, err
	}

	return resource, nil
}

func (p *Postgres) ListResources(ctx context.Context) ([]Resource, error) {
	query := `
		SELECT id::text, name, type, COALESCE(description, ''), created_at
		FROM resources
		ORDER BY created_at DESC
	`

	rows, err := p.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]Resource, 0)

	for rows.Next() {
		var resource Resource

		err := rows.Scan(
			&resource.ID,
			&resource.Name,
			&resource.Type,
			&resource.Description,
			&resource.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		resources = append(resources, resource)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resources, nil
}
