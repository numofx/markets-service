// Package pricing centralizes instrument-aware price conversions.
//
// Variance is canonical. Volatility is display-only.
//
// For BTCVAR30-PERP:
//   - submitted and matched prices are variance prices
//   - internal executor prices are variance ticks
//   - displayed vol percent is derived only at presentation boundaries
package pricing
