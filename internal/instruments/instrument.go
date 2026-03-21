package instruments

import "time"

const (
	PricingModelLinear   = "linear"
	PricingModelVariance = "variance"
	PricingModelVol      = "volatility"

	DisplayPriceDirect     = "direct"
	DisplayPriceVolPercent = "vol_percent"
)

type Metadata struct {
	Symbol             string        `json:"symbol"`
	AssetAddress       string        `json:"asset_address"`
	SubID              string        `json:"sub_id"`
	TickSize           string        `json:"tick_size"`
	MinSize            string        `json:"min_size"`
	ContractMultiplier string        `json:"contract_multiplier"`
	QuotePrecision     int           `json:"quote_precision"`
	PricingModel       string        `json:"pricing_model,omitempty"`
	PriceSemantics     string        `json:"price_semantics,omitempty"`
	DisplayPriceKind   string        `json:"display_price_kind,omitempty"`
	DisplaySemantics   string        `json:"display_semantics,omitempty"`
	DisplayLabel       string        `json:"display_label,omitempty"`
	DisplayName        string        `json:"display_name,omitempty"`
	SettlementNote     string        `json:"settlement_note,omitempty"`
	FundingInterval    time.Duration `json:"-"`
	Enabled            bool          `json:"enabled"`
}

type Registry struct {
	bySymbol       map[string]Metadata
	byAssetAddress map[string]Metadata
}

func NewRegistry(items []Metadata) *Registry {
	registry := &Registry{
		bySymbol:       make(map[string]Metadata, len(items)),
		byAssetAddress: make(map[string]Metadata, len(items)),
	}

	for _, item := range items {
		registry.bySymbol[item.Symbol] = item
		if item.AssetAddress != "" {
			registry.byAssetAddress[item.AssetAddress] = item
		}
	}

	return registry
}

func (r *Registry) Enabled() []Metadata {
	if r == nil {
		return nil
	}

	items := make([]Metadata, 0, len(r.bySymbol))
	for _, item := range r.bySymbol {
		if item.Enabled {
			items = append(items, item)
		}
	}
	return items
}

func (r *Registry) BySymbol(symbol string) (Metadata, bool) {
	if r == nil {
		return Metadata{}, false
	}
	item, ok := r.bySymbol[symbol]
	return item, ok
}

func (r *Registry) ByAssetAddress(assetAddress string) (Metadata, bool) {
	if r == nil {
		return Metadata{}, false
	}
	item, ok := r.byAssetAddress[assetAddress]
	return item, ok
}
