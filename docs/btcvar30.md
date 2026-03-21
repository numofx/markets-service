# BTCVAR30 Architecture Note

## Overview

The v1 BTCVAR30 implementation stays inside `matching-backend` and splits into four small pieces:

1. `internal/marketdata/deribit`
   Deribit JSON-RPC client with typed methods for volatility index data, instruments, and order books.
2. `internal/oracles/btcvar30`
   Oracle derivation, signing abstraction, in-memory latest cache, and Postgres history persistence.
3. `internal/instruments`
   Instrument metadata registry including `BTCVAR30-PERP`.
4. `internal/funding`
   Funding loop that reads the latest oracle payload and current book midpoint mark.

## Data Flow

1. `cmd/api` starts the Deribit client, BTCVAR30 oracle service, and BTCVAR30 funding loop.
2. The oracle service polls Deribit BTC volatility index candles.
3. The latest candle close becomes `vol_30d`.
4. The backend derives:

```text
variance_30d = (vol_30d / 100)^2
```

5. The canonical payload is deterministically signed, cached in memory, and persisted in `oracle_btcvar30_history`.
6. Public routes serve the latest payload and history.
7. The funding loop reads the latest payload, computes a conservative midpoint mark from the BTCVAR30 order book, and derives funding.

## Safety

- If Deribit is unavailable, the last payload remains available but naturally becomes stale.
- If the oracle is stale, funding pauses.
- Logs are emitted on oracle success, oracle failure, stale state, funding calculations, and disabled instrument startup.
- The signer is intentionally minimal. If a repo-wide signing scheme appears later, replace `DeterministicSigner`.

## Known Limitations

- v1 uses Deribit volatility index data directly.
- v1 does not build an options-surface-derived variance index.
- This repo does not yet contain a full risk engine, so position or leverage caps are metadata-only follow-up work.
