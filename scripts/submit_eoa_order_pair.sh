#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://127.0.0.1:8080}"
OWNER_ADDRESS="${OWNER_ADDRESS:-0xC7bE60b228b997c23094DdfdD71e22E2DE6C9310}"
SIGNER_ADDRESS="${SIGNER_ADDRESS:-$OWNER_ADDRESS}"
TRADE_MODULE_ADDRESS="${TRADE_MODULE_ADDRESS:-0x5fba217bFf9DfE7EDaD333972866DbA83c50B0f2}"
BTC_PERP_ASSET_ADDRESS="${BTC_PERP_ASSET_ADDRESS:-}"

TAKER_SUBACCOUNT_ID="${TAKER_SUBACCOUNT_ID:-10}"
MAKER_SUBACCOUNT_ID="${MAKER_SUBACCOUNT_ID:-11}"
TAKER_NONCE="${TAKER_NONCE:-1}"
MAKER_NONCE="${MAKER_NONCE:-2}"
EXPIRY="${EXPIRY:-1893456000}"

TAKER_ORDER_ID="${TAKER_ORDER_ID:-taker-eoa-1}"
MAKER_ORDER_ID="${MAKER_ORDER_ID:-maker-eoa-1}"

TAKER_SIDE="${TAKER_SIDE:-buy}"
MAKER_SIDE="${MAKER_SIDE:-sell}"
TAKER_LIMIT_PRICE="${TAKER_LIMIT_PRICE:-100}"
MAKER_LIMIT_PRICE="${MAKER_LIMIT_PRICE:-90}"
DESIRED_AMOUNT="${DESIRED_AMOUNT:-1000000000000000000}"
WORST_FEE="${WORST_FEE:-0}"

TAKER_ACTION_DATA="${TAKER_ACTION_DATA:?set TAKER_ACTION_DATA to the signed trade-module calldata hex}"
MAKER_ACTION_DATA="${MAKER_ACTION_DATA:?set MAKER_ACTION_DATA to the signed trade-module calldata hex}"
TAKER_SIGNATURE="${TAKER_SIGNATURE:?set TAKER_SIGNATURE to the taker action signature hex}"
MAKER_SIGNATURE="${MAKER_SIGNATURE:?set MAKER_SIGNATURE to the maker action signature hex}"

if [ -z "$BTC_PERP_ASSET_ADDRESS" ]; then
  echo "BTC_PERP_ASSET_ADDRESS must be set" >&2
  exit 1
fi

submit_order() {
  local order_id="$1"
  local subaccount_id="$2"
  local nonce="$3"
  local side="$4"
  local limit_price="$5"
  local action_data="$6"
  local signature="$7"

  local payload
  payload=$(cat <<JSON
{
  "order_id": "$order_id",
  "owner_address": "$OWNER_ADDRESS",
  "signer_address": "$SIGNER_ADDRESS",
  "subaccount_id": "$subaccount_id",
  "recipient_id": "$subaccount_id",
  "nonce": "$nonce",
  "side": "$side",
  "asset_address": "$BTC_PERP_ASSET_ADDRESS",
  "sub_id": "0",
  "desired_amount": "$DESIRED_AMOUNT",
  "filled_amount": "0",
  "limit_price": "$limit_price",
  "worst_fee": "$WORST_FEE",
  "expiry": $EXPIRY,
  "action_json": {
    "subaccount_id": "$subaccount_id",
    "nonce": "$nonce",
    "module": "$TRADE_MODULE_ADDRESS",
    "data": "$action_data",
    "expiry": "$EXPIRY",
    "owner": "$OWNER_ADDRESS",
    "signer": "$SIGNER_ADDRESS"
  },
  "signature": "$signature"
}
JSON
)

  curl -sS -X POST "$API_URL/v1/orders" \
    -H 'accept: application/json' \
    -H 'content-type: application/json' \
    --data "$payload"
  echo
}

submit_order "$TAKER_ORDER_ID" "$TAKER_SUBACCOUNT_ID" "$TAKER_NONCE" "$TAKER_SIDE" "$TAKER_LIMIT_PRICE" "$TAKER_ACTION_DATA" "$TAKER_SIGNATURE"
submit_order "$MAKER_ORDER_ID" "$MAKER_SUBACCOUNT_ID" "$MAKER_NONCE" "$MAKER_SIDE" "$MAKER_LIMIT_PRICE" "$MAKER_ACTION_DATA" "$MAKER_SIGNATURE"
