package btcvar30

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, payload Payload) error {
	const query = `
insert into oracle_btcvar30_history (
  symbol,
  source,
  observed_at,
  vol_30d,
  variance_30d,
  methodology_version,
  signature
) values ($1, $2, $3, $4, $5, $6, $7)
on conflict (symbol, observed_at) do update
set source = excluded.source,
    vol_30d = excluded.vol_30d,
    variance_30d = excluded.variance_30d,
    methodology_version = excluded.methodology_version,
    signature = excluded.signature
`

	_, err := r.pool.Exec(
		ctx,
		query,
		payload.Symbol,
		payload.Source,
		payload.Timestamp.UTC(),
		payload.Vol30D,
		payload.Variance30D,
		payload.MethodologyVersion,
		payload.Signature,
	)
	return err
}

func (r *Repository) Latest(ctx context.Context) (Payload, error) {
	const query = `
select symbol, source, observed_at, vol_30d, variance_30d, methodology_version, signature
from oracle_btcvar30_history
order by observed_at desc
limit 1
`

	var payload Payload
	if err := r.pool.QueryRow(ctx, query).Scan(
		&payload.Symbol,
		&payload.Source,
		&payload.Timestamp,
		&payload.Vol30D,
		&payload.Variance30D,
		&payload.MethodologyVersion,
		&payload.Signature,
	); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func (r *Repository) History(ctx context.Context, limit int) ([]Payload, error) {
	if limit <= 0 {
		limit = 100
	}

	const query = `
select symbol, source, observed_at, vol_30d, variance_30d, methodology_version, signature
from oracle_btcvar30_history
order by observed_at desc
limit $1
`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Payload
	for rows.Next() {
		var payload Payload
		if err := rows.Scan(
			&payload.Symbol,
			&payload.Source,
			&payload.Timestamp,
			&payload.Vol30D,
			&payload.Variance30D,
			&payload.MethodologyVersion,
			&payload.Signature,
		); err != nil {
			return nil, err
		}
		items = append(items, payload)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) Require() error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("btcvar30 repository is not configured")
	}
	return nil
}
