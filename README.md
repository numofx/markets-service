# matching-backend

Thin offchain backend for the `matching` contracts.

Initial scope:

- one market: BTC perp
- one order type: limit order
- one module path: `TradeModule`
- one executor
- one database table for active orders
- one matching loop

This repo is intentionally narrow. It is not a generic exchange backend.

## Responsibilities

- accept and persist signed BTC perp orders
- expose a minimal API for order entry and book inspection
- run a price-time matching loop
- submit executor payloads for `Matching.verifyAndMatch(...)`

## Out of Scope

- RFQ
- liquidation
- multi-market support
- websocket market data
- a full frontend
- direct onchain execution from Go

## Layout

```text
cmd/
  api/        HTTP API for orders and health checks
  matcher/    background matching worker
internal/
  api/        HTTP server wiring and handlers
  config/     environment configuration
  db/         Postgres connection helpers
  matching/   matching loop and orchestration
  orders/     order model and repository contracts
migrations/   database schema
```

## Configuration

Copy `.env.example` into your own environment and set the required values.

Important values:

- `DATABASE_URL`
- `API_ADDR`
- `MATCHER_POLL_INTERVAL`
- `CHAIN_ID`
- `MATCHING_ADDRESS`
- `TRADE_MODULE_ADDRESS`
- `BTC_PERP_ASSET_ADDRESS`
- optionally `EXPECTED_ORDER_OWNER`
- optionally `EXPECTED_ORDER_SIGNER`
- `EXECUTOR_URL`
- optionally `EXECUTOR_MANAGER_DATA`
- optionally `EXECUTOR_MANAGER_DATA_FILE`

If `EXPECTED_ORDER_OWNER` or `EXPECTED_ORDER_SIGNER` are set, the API rejects orders whose declared owner/signer do not match those configured addresses. The API also validates that `action_json.owner`, `action_json.signer`, `action_json.subaccount_id`, and `action_json.nonce` match the stored order fields.

`EXECUTOR_URL` is the endpoint for a separate executor process, likely implemented in
TypeScript with `viem`, that performs simulation and submits `verifyAndMatch(...)`.

`EXECUTOR_MANAGER_DATA` lets the matcher attach the exact `manager_data` hex required by the
executor call. If the blob is too large for an env var, set `EXECUTOR_MANAGER_DATA_FILE`
instead. That file may contain either the raw hex string or a JSON object with a
`manager_data` field.

Expected request body:

```json
{
  "market": "BTC-PERP",
  "asset_address": "0x...",
  "module_address": "0x...",
  "maker_order_id": "maker-order-id",
  "taker_order_id": "taker-order-id",
  "actions": [
    {
      "subaccount_id": "123",
      "nonce": "1",
      "module": "0x...",
      "data": "0x...",
      "expiry": "1710000000",
      "owner": "0x...",
      "signer": "0x..."
    }
  ],
  "signatures": ["0x..."],
  "order_data": {
    "taker_account": "123",
    "taker_fee": "0",
    "fill_details": [
      {
        "filled_account": "456",
        "amount_filled": "1000000000000000000",
        "price": "78000000000000000000",
        "fee": "0"
      }
    ],
    "manager_data": "0x..."
  }
}
```

The executor may return an empty `2xx` response or JSON like:

```json
{
  "accepted": true,
  "tx_hash": "0x..."
}
```

## Development

Expected local stack:

- Go 1.24+
- PostgreSQL 16+

Suggested flow:

1. Start Postgres.
2. Apply migrations from `migrations/`.
3. Run the API:

```bash
go run ./cmd/api
```

4. Run the matcher:

```bash
go run ./cmd/matcher
```

### EOA-Owned Order Submission

For an EOA-owned deployment, set:

```dotenv
EXPECTED_ORDER_OWNER=0xC7bE60b228b997c23094DdfdD71e22E2DE6C9310
EXPECTED_ORDER_SIGNER=0xC7bE60b228b997c23094DdfdD71e22E2DE6C9310
```

Then submit orders whose top-level fields and `action_json` agree on:

- `owner_address` / `action_json.owner`
- `signer_address` / `action_json.signer`
- `subaccount_id` / `action_json.subaccount_id`
- `nonce` / `action_json.nonce`

Example EOA-owned order templates are in:

- [examples/eoa_taker_order.json](/Users/robertleifke/Code/work/matching-backend/examples/eoa_taker_order.json)
- [examples/eoa_maker_order.json](/Users/robertleifke/Code/work/matching-backend/examples/eoa_maker_order.json)

A helper script is available at:

- [scripts/submit_eoa_order_pair.sh](/Users/robertleifke/Code/work/matching-backend/scripts/submit_eoa_order_pair.sh)

It posts a crossed taker/maker pair to `/v1/orders`, but you still need to provide real `TAKER_ACTION_DATA`, `MAKER_ACTION_DATA`, `TAKER_SIGNATURE`, and `MAKER_SIGNATURE` values for the orders to execute successfully through the onchain matcher.

To reproduce the verified Base dry-run path for BTC squared, point the backend at the generated
manager data file from the executor repo:

```dotenv
EXECUTOR_MANAGER_DATA_FILE=/tmp/perp-manager-data.json
```

That file can be generated with:

- [generate_perp_manager_data.mjs](/Users/robertleifke/Code/work/matching-executor/scripts/generate_perp_manager_data.mjs)

The matcher will then forward the `manager_data` blob automatically in every executor payload
instead of hardcoding `0x`.

## First Milestone

The first milestone is one successful matched BTC perp trade through `TradeModule`:

1. store two signed crossed orders
2. match them offchain
3. produce executor payloads from stored signed actions
4. send them to the executor
5. update both orders on success
