package instruments

import (
	"strings"
	"time"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/instruments/btcvar30"
)

const (
	BTCConvexPerpSymbol = "BTCUSDC-CVXPERP"
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
	}

	return NewRegistry(items)
}
