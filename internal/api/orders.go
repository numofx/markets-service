package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/orders"
)

type createOrderRequest struct {
	OrderID       string          `json:"order_id"`
	OwnerAddress  string          `json:"owner_address"`
	SignerAddress string          `json:"signer_address"`
	SubaccountID  string          `json:"subaccount_id"`
	RecipientID   string          `json:"recipient_id"`
	Nonce         string          `json:"nonce"`
	Side          string          `json:"side"`
	AssetAddress  string          `json:"asset_address"`
	SubID         string          `json:"sub_id"`
	DesiredAmount string          `json:"desired_amount"`
	FilledAmount  string          `json:"filled_amount"`
	LimitPrice    string          `json:"limit_price"`
	WorstFee      string          `json:"worst_fee"`
	Expiry        int64           `json:"expiry"`
	ActionJSON    json.RawMessage `json:"action_json"`
	Signature     string          `json:"signature"`
}

func (r createOrderRequest) toParams(cfg config.Config) (orders.CreateOrderParams, error) {
	if r.OrderID == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("order_id is required")
	}
	if r.OwnerAddress == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("owner_address is required")
	}
	if r.SignerAddress == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("signer_address is required")
	}
	if r.SubaccountID == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("subaccount_id is required")
	}
	if r.RecipientID == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("recipient_id is required")
	}
	if r.Nonce == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("nonce is required")
	}
	if r.AssetAddress == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("asset_address is required")
	}
	if r.DesiredAmount == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("desired_amount is required")
	}
	if r.LimitPrice == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("limit_price is required")
	}
	if r.WorstFee == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("worst_fee is required")
	}
	if r.Signature == "" {
		return orders.CreateOrderParams{}, fmt.Errorf("signature is required")
	}
	if r.Expiry <= time.Now().Unix() {
		return orders.CreateOrderParams{}, fmt.Errorf("expiry must be in the future")
	}
	if !json.Valid(r.ActionJSON) {
		return orders.CreateOrderParams{}, fmt.Errorf("action_json must be valid JSON")
	}

	side := orders.Side(strings.ToLower(r.Side))
	if side != orders.SideBuy && side != orders.SideSell {
		return orders.CreateOrderParams{}, fmt.Errorf("side must be 'buy' or 'sell'")
	}

	assetAddress := strings.ToLower(r.AssetAddress)
	if strings.ToLower(cfg.BTCPerpAssetAddress) != "" && assetAddress != strings.ToLower(cfg.BTCPerpAssetAddress) {
		return orders.CreateOrderParams{}, fmt.Errorf("asset_address must match configured BTC perp asset")
	}

	subID := r.SubID
	if subID == "" {
		subID = "0"
	}

	filledAmount := r.FilledAmount
	if filledAmount == "" {
		filledAmount = "0"
	}

	return orders.CreateOrderParams{
		OrderID:       r.OrderID,
		OwnerAddress:  strings.ToLower(r.OwnerAddress),
		SignerAddress: strings.ToLower(r.SignerAddress),
		SubaccountID:  r.SubaccountID,
		RecipientID:   r.RecipientID,
		Nonce:         r.Nonce,
		Side:          side,
		AssetAddress:  assetAddress,
		SubID:         subID,
		DesiredAmount: r.DesiredAmount,
		FilledAmount:  filledAmount,
		LimitPrice:    r.LimitPrice,
		WorstFee:      r.WorstFee,
		Expiry:        r.Expiry,
		ActionJSON:    r.ActionJSON,
		Signature:     r.Signature,
	}, nil
}

type cancelOrderRequest struct {
	OwnerAddress string `json:"owner_address"`
	Nonce        string `json:"nonce"`
}

func (r cancelOrderRequest) validate() error {
	if r.OwnerAddress == "" {
		return fmt.Errorf("owner_address is required")
	}
	if r.Nonce == "" {
		return fmt.Errorf("nonce is required")
	}
	return nil
}
