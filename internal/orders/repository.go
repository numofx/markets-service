package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type CreateOrderParams struct {
	OrderID       string
	OwnerAddress  string
	SignerAddress string
	SubaccountID  string
	RecipientID   string
	Nonce         string
	Side          Side
	AssetAddress  string
	SubID         string
	DesiredAmount string
	FilledAmount  string
	LimitPrice    string
	WorstFee      string
	Expiry        int64
	ActionJSON    json.RawMessage
	Signature     string
}

type CancelOrderParams struct {
	OwnerAddress string
	Nonce        string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, params CreateOrderParams) (Order, error) {
	const query = `
insert into active_orders (
  order_id,
  owner_address,
  signer_address,
  subaccount_id,
  recipient_id,
  nonce,
  side,
  asset_address,
  sub_id,
  desired_amount,
  filled_amount,
  limit_price,
  worst_fee,
  expiry,
  action_json,
  signature,
  status
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
returning order_id, owner_address, signer_address, subaccount_id, recipient_id, nonce, side, asset_address, sub_id,
          desired_amount, filled_amount, limit_price, worst_fee, expiry, action_json, signature, status, created_at
`

	order := Order{}
	if err := r.pool.QueryRow(
		ctx,
		query,
		params.OrderID,
		params.OwnerAddress,
		params.SignerAddress,
		params.SubaccountID,
		params.RecipientID,
		params.Nonce,
		params.Side,
		params.AssetAddress,
		params.SubID,
		params.DesiredAmount,
		params.FilledAmount,
		params.LimitPrice,
		params.WorstFee,
		params.Expiry,
		params.ActionJSON,
		params.Signature,
		StatusActive,
	).Scan(
		&order.OrderID,
		&order.OwnerAddress,
		&order.SignerAddress,
		&order.SubaccountID,
		&order.RecipientID,
		&order.Nonce,
		&order.Side,
		&order.AssetAddress,
		&order.SubID,
		&order.DesiredAmount,
		&order.FilledAmount,
		&order.LimitPrice,
		&order.WorstFee,
		&order.Expiry,
		&order.ActionJSON,
		&order.Signature,
		&order.Status,
		&order.CreatedAt,
	); err != nil {
		return Order{}, mapPGError(err)
	}

	return order, nil
}

func (r *Repository) CancelByOwnerNonce(ctx context.Context, params CancelOrderParams) (Order, error) {
	const query = `
update active_orders
set status = $3
where owner_address = $1 and nonce = $2 and status = 'active'
returning order_id, owner_address, signer_address, subaccount_id, recipient_id, nonce, side, asset_address, sub_id,
          desired_amount, filled_amount, limit_price, worst_fee, expiry, action_json, signature, status, created_at
`

	order := Order{}
	if err := r.pool.QueryRow(ctx, query, params.OwnerAddress, params.Nonce, StatusCancelled).Scan(
		&order.OrderID,
		&order.OwnerAddress,
		&order.SignerAddress,
		&order.SubaccountID,
		&order.RecipientID,
		&order.Nonce,
		&order.Side,
		&order.AssetAddress,
		&order.SubID,
		&order.DesiredAmount,
		&order.FilledAmount,
		&order.LimitPrice,
		&order.WorstFee,
		&order.Expiry,
		&order.ActionJSON,
		&order.Signature,
		&order.Status,
		&order.CreatedAt,
	); err != nil {
		return Order{}, mapPGError(err)
	}

	return order, nil
}

func (r *Repository) ListBook(ctx context.Context, assetAddress string, subID string, limit int32) ([]Order, []Order, error) {
	bids, err := r.listBySide(ctx, assetAddress, subID, SideBuy, limit)
	if err != nil {
		return nil, nil, err
	}

	asks, err := r.listBySide(ctx, assetAddress, subID, SideSell, limit)
	if err != nil {
		return nil, nil, err
	}

	return bids, asks, nil
}

func (r *Repository) BestBidAndAsk(ctx context.Context, assetAddress string, subID string) (*Order, *Order, error) {
	bid, err := r.bestBySide(ctx, assetAddress, subID, SideBuy)
	if err != nil {
		return nil, nil, err
	}

	ask, err := r.bestBySide(ctx, assetAddress, subID, SideSell)
	if err != nil {
		return nil, nil, err
	}

	return bid, ask, nil
}

func (r *Repository) listBySide(ctx context.Context, assetAddress string, subID string, side Side, limit int32) ([]Order, error) {
	orderBy := "limit_price desc, created_at asc"
	if side == SideSell {
		orderBy = "limit_price asc, created_at asc"
	}

	query := fmt.Sprintf(`
select order_id, owner_address, signer_address, subaccount_id, recipient_id, nonce, side, asset_address, sub_id,
       desired_amount, filled_amount, limit_price, worst_fee, expiry, action_json, signature, status, created_at
from active_orders
where asset_address = $1 and sub_id = $2 and side = $3 and status = 'active'
order by %s
limit $4
`, orderBy)

	rows, err := r.pool.Query(ctx, query, assetAddress, subID, side, limit)
	if err != nil {
		return nil, mapPGError(err)
	}
	defer rows.Close()

	var results []Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, order)
	}

	if err := rows.Err(); err != nil {
		return nil, mapPGError(err)
	}

	return results, nil
}

func (r *Repository) bestBySide(ctx context.Context, assetAddress string, subID string, side Side) (*Order, error) {
	orders, err := r.listBySide(ctx, assetAddress, subID, side, 1)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, nil
	}
	return &orders[0], nil
}

func scanOrder(row pgx.Row) (Order, error) {
	var order Order
	if err := row.Scan(
		&order.OrderID,
		&order.OwnerAddress,
		&order.SignerAddress,
		&order.SubaccountID,
		&order.RecipientID,
		&order.Nonce,
		&order.Side,
		&order.AssetAddress,
		&order.SubID,
		&order.DesiredAmount,
		&order.FilledAmount,
		&order.LimitPrice,
		&order.WorstFee,
		&order.Expiry,
		&order.ActionJSON,
		&order.Signature,
		&order.Status,
		&order.CreatedAt,
	); err != nil {
		return Order{}, mapPGError(err)
	}

	return order, nil
}

func mapPGError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("duplicate order: %w", err)
	}

	return err
}

func (o Order) IsExpired(now time.Time) bool {
	return o.Expiry <= now.Unix()
}
