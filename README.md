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
- produce `MatchInstruction` payloads for an executor

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
- `EXECUTOR_URL`

`EXECUTOR_URL` is the endpoint for a separate executor process, likely implemented in
TypeScript with `viem`, that performs simulation and submits `verifyAndMatch(...)`.

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

## First Milestone

The first milestone is one successful matched BTC perp trade through `TradeModule`:

1. store two signed crossed orders
2. match them offchain
3. produce one `MatchInstruction`
4. send it to the executor
5. mark both orders as filled on success
