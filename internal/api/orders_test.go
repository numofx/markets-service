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
