package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/numofx/matching-backend/internal/config"
)

func TestCreateOrderRequestToParamsRejectsActionJSONOwnerMismatch(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xdef",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xasset",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "75",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xwrong","signer":"0xdef"}`),
		Signature:     "0xsig",
	}

	_, err := req.toParams(config.Config{})
	if err == nil || err.Error() != "action_json.owner must match owner_address" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateOrderRequestToParamsRejectsUnexpectedConfiguredSigner(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xdef",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xasset",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "75",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xabc","signer":"0xdef"}`),
		Signature:     "0xsig",
	}

	_, err := req.toParams(config.Config{ExpectedOrderSigner: "0x123"})
	if err == nil || err.Error() != "signer_address must match configured expected signer" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateOrderRequestToParamsParsesBTCVar30DecimalPrice(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xabc",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xvar",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "0.2724",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xabc","signer":"0xabc"}`),
		Signature:     "0xsig",
	}

	params, err := req.toParams(config.Config{
		BTCVar30Enabled:      true,
		BTCVar30AssetAddress: "0xvar",
	})
	if err != nil {
		t.Fatalf("toParams returned error: %v", err)
	}
	if params.LimitPrice != "0.2724" {
		t.Fatalf("display limit price = %s", params.LimitPrice)
	}
	if params.LimitPriceTicks != "2724" {
		t.Fatalf("limit price ticks = %s", params.LimitPriceTicks)
	}
}

func TestCreateOrderRequestToParamsRejectsOffTickBTCVar30Price(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xabc",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xvar",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "0.27245",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xabc","signer":"0xabc"}`),
		Signature:     "0xsig",
	}

	_, err := req.toParams(config.Config{
		BTCVar30Enabled:      true,
		BTCVar30AssetAddress: "0xvar",
	})
	if err == nil || err.Error() != "price must align to tick size 0.0001" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateOrderRequestToParamsRejectsVolLookingBTCVar30Price(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xabc",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xvar",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "52.0",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xabc","signer":"0xabc"}`),
		Signature:     "0xsig",
	}

	_, err := req.toParams(config.Config{
		BTCVar30Enabled:      true,
		BTCVar30AssetAddress: "0xvar",
	})
	if err == nil || err.Error() != "BTCVAR30 prices are variance, not volatility. Example: 0.25 = 50% vol" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateOrderRequestToParamsAcceptsVarianceLookingBTCVar30Price(t *testing.T) {
	req := createOrderRequest{
		OrderID:       "order-1",
		OwnerAddress:  "0xabc",
		SignerAddress: "0xabc",
		SubaccountID:  "10",
		RecipientID:   "10",
		Nonce:         "1",
		Side:          "buy",
		AssetAddress:  "0xvar",
		SubID:         "0",
		DesiredAmount: "100",
		FilledAmount:  "0",
		LimitPrice:    "0.52",
		WorstFee:      "1",
		Expiry:        time.Now().Add(time.Hour).Unix(),
		ActionJSON:    json.RawMessage(`{"subaccount_id":"10","nonce":"1","module":"0xtrade","data":"0xaaa","expiry":"100","owner":"0xabc","signer":"0xabc"}`),
		Signature:     "0xsig",
	}

	params, err := req.toParams(config.Config{
		BTCVar30Enabled:      true,
		BTCVar30AssetAddress: "0xvar",
	})
	if err != nil {
		t.Fatalf("toParams returned error: %v", err)
	}
	if params.LimitPrice != "0.52" || params.LimitPriceTicks != "5200" {
		t.Fatalf("unexpected params %+v", params)
	}
}
