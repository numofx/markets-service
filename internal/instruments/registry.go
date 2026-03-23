package instruments

import (
	"strings"
	"time"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/instruments/btcvar30"
)

const (
	BTCConvexPerpSymbol = "BTCUSDC-CVXPERP"
	CNGNApr2026Symbol   = "USDC/cNGN-APR30-2026"
)

func DefaultRegistry(cfg config.Config) *Registry {
	items := []Metadata{
		{
			Symbol:             BTCConvexPerpSymbol,
			AssetAddress:       strings.ToLower(strings.TrimSpace(cfg.BTCPerpAssetAddress)),
			SubID:              "0",
			TickSize:           "1",
			MinSize:            "1",
			ContractMultiplier: "1",
			QuotePrecision:     8,
			PricingModel:       PricingModelLinear,
			PriceSemantics:     PricingModelLinear,
			DisplayPriceKind:   DisplayPriceDirect,
			DisplaySemantics:   DisplayPriceDirect,
			DisplayName:        "BTC Convex Perpetual",
			FundingInterval:    8 * time.Hour,
			Enabled:            strings.TrimSpace(cfg.BTCPerpAssetAddress) != "",
		},
		{
			Symbol:             btcvar30.Symbol,
			AssetAddress:       strings.ToLower(strings.TrimSpace(cfg.BTCVar30AssetAddress)),
			SubID:              btcvar30.SubID,
			TickSize:           btcvar30.TickSize,
			MinSize:            btcvar30.MinSize,
			ContractMultiplier: btcvar30.ContractMultiplier,
			QuotePrecision:     btcvar30.QuotePrecision,
			PricingModel:       PricingModelVariance,
			PriceSemantics:     PricingModelVariance,
			DisplayPriceKind:   DisplayPriceVolPercent,
			DisplaySemantics:   DisplayPriceVolPercent,
			DisplayLabel:       btcvar30.DisplayLabel,
			DisplayName:        btcvar30.DisplayName,
			SettlementNote:     btcvar30.SettlementNote,
			FundingInterval:    cfg.BTCVar30FundingInterval,
			Enabled:            cfg.BTCVar30Enabled,
		},
		{
			Symbol:             CNGNApr2026Symbol,
			AssetAddress:       strings.ToLower(strings.TrimSpace(cfg.CNGNApr2026FutureAssetAddress)),
			SubID:              strings.TrimSpace(cfg.CNGNApr2026FutureSubID),
			ContractType:       "deliverable_fx_future",
			SettlementType:     "physical_delivery",
			BaseAssetSymbol:    "USDC",
			QuoteAssetSymbol:   "cNGN",
			ExpiryTimestamp:    1777507200,
			LastTradeTimestamp: 1777420800,
			TickSize:           "1",
			MinSize:            "0.001",
			ContractMultiplier: "10000",
			QuotePrecision:     6,
			PricingModel:       PricingModelLinear,
			PriceSemantics:     PricingModelLinear,
			DisplayPriceKind:   DisplayPriceDirect,
			DisplaySemantics:   DisplayPriceDirect,
			DisplayLabel:       "cNGN per USDC",
			DisplayName:        "USDC/cNGN APR-30-2026 Future",
			SettlementNote:     "Physically delivered on Base. Long pays cNGN and receives fixed USDC notional; short pays fixed USDC notional and receives cNGN.",
			Enabled:            strings.TrimSpace(cfg.CNGNApr2026FutureAssetAddress) != "" && strings.TrimSpace(cfg.CNGNApr2026FutureSubID) != "",
		},
	}

	return NewRegistry(items)
}
