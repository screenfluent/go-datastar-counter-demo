package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/szymon/go-datastar-counter-demo/internal/counter"
	"github.com/szymon/go-datastar-counter-demo/internal/db"
)

type Postgres struct {
	queries *db.Queries
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{queries: db.New(pool)}
}

func (p *Postgres) Snapshot(ctx context.Context) (counter.Snapshot, error) {
	row, err := p.queries.GetCounter(ctx)
	if err != nil {
		return counter.Snapshot{}, err
	}
	return counter.Snapshot{
		Value:     row.Value,
		UpdatedAt: row.UpdatedAt.Time,
		Source:    "postgres",
	}, nil
}

func (p *Postgres) Apply(ctx context.Context, delta int) (counter.Snapshot, error) {
	row, err := p.queries.ChangeCounter(ctx, int32(delta))
	if err != nil {
		snapshot, snapErr := p.Snapshot(ctx)
		if snapErr != nil {
			return counter.Snapshot{}, err
		}
		snapshot.Error = "licznik nie moze spasc ponizej zera"
		return snapshot, err
	}
	return counter.Snapshot{
		Value:     row.Value,
		UpdatedAt: row.UpdatedAt.Time,
		Source:    "postgres",
	}, nil
}

func (p *Postgres) Reset(ctx context.Context) (counter.Snapshot, error) {
	row, err := p.queries.ResetCounter(ctx)
	if err != nil {
		return counter.Snapshot{}, err
	}
	return counter.Snapshot{
		Value:     row.Value,
		UpdatedAt: row.UpdatedAt.Time,
		Source:    "postgres",
	}, nil
}
