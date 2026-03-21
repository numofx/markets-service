package matching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/numofx/matching-backend/internal/orders"
)

type ExecutorClient struct {
	url         string
	managerData string
	httpClient  *http.Client
}

type ExecutorRequest struct {
	Market        string            `json:"market"`
	AssetAddress  string            `json:"asset_address"`
	ModuleAddress string            `json:"module_address"`
	MakerOrderID  string            `json:"maker_order_id"`
	TakerOrderID  string            `json:"taker_order_id"`
	Actions       []json.RawMessage `json:"actions"`
	Signatures    []string          `json:"signatures"`
	OrderData     TradeOrderData    `json:"order_data"`
}

type TradeOrderData struct {
	TakerAccount string            `json:"taker_account"`
	TakerFee     string            `json:"taker_fee"`
	FillDetails  []TradeFillDetail `json:"fill_details"`
	ManagerData  string            `json:"manager_data"`
}

type TradeFillDetail struct {
	FilledAccount string `json:"filled_account"`
	AmountFilled  string `json:"amount_filled"`
	// Price is always the canonical internal price. For BTCVAR30-PERP that means
	// variance-price ticks, not vol points.
	Price string `json:"price"`
	Fee   string `json:"fee"`
}

type ExecutorResponse struct {
	Accepted bool   `json:"accepted"`
	TxHash   string `json:"tx_hash"`
	Error    string `json:"error"`
}

func NewExecutorClient(url string, managerData string) *ExecutorClient {
	return &ExecutorClient{
		url:         strings.TrimSpace(url),
		managerData: strings.TrimSpace(managerData),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *ExecutorClient) SubmitMatch(ctx context.Context, candidate orders.MatchCandidate, price string, amount string) (ExecutorResponse, error) {
	return c.SubmitMatchForMarket(ctx, "BTCUSDC-CVXPERP", candidate, price, amount)
}

func (c *ExecutorClient) SubmitMatchForMarket(ctx context.Context, market string, candidate orders.MatchCandidate, price string, amount string) (ExecutorResponse, error) {
	if c.url == "" {
		return ExecutorResponse{}, fmt.Errorf("EXECUTOR_URL is required")
	}

	reqBody, err := buildExecutorRequest(market, candidate, c.managerData, price, amount)
	if err != nil {
		return ExecutorResponse{}, err
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return ExecutorResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return ExecutorResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ExecutorResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ExecutorResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(body) == 0 {
			return ExecutorResponse{}, fmt.Errorf("executor returned status %d", resp.StatusCode)
		}
		return ExecutorResponse{}, fmt.Errorf("executor returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return ExecutorResponse{Accepted: true}, nil
	}

	var executorResp ExecutorResponse
	if err := json.Unmarshal(body, &executorResp); err != nil {
		return ExecutorResponse{}, fmt.Errorf("decode executor response: %w", err)
	}
	if executorResp.Error != "" {
		return ExecutorResponse{}, fmt.Errorf("executor rejected match: %s", executorResp.Error)
	}
	if !executorResp.Accepted && executorResp.TxHash == "" {
		executorResp.Accepted = true
	}
	return executorResp, nil
}

func buildExecutorRequest(market string, candidate orders.MatchCandidate, managerData string, price string, amount string) (ExecutorRequest, error) {
	takerAction, err := normalizeAction(candidate.Taker)
	if err != nil {
		return ExecutorRequest{}, fmt.Errorf("parse taker action_json: %w", err)
	}
	makerAction, err := normalizeAction(candidate.Maker)
	if err != nil {
		return ExecutorRequest{}, fmt.Errorf("parse maker action_json: %w", err)
	}

	return ExecutorRequest{
		Market:        market,
		AssetAddress:  strings.ToLower(candidate.Taker.AssetAddress),
		ModuleAddress: extractModuleAddress(takerAction),
		MakerOrderID:  candidate.Maker.OrderID,
		TakerOrderID:  candidate.Taker.OrderID,
		Actions:       []json.RawMessage{takerAction, makerAction},
		Signatures:    []string{candidate.Taker.Signature, candidate.Maker.Signature},
		OrderData: TradeOrderData{
			TakerAccount: candidate.Taker.SubaccountID,
			TakerFee:     "0",
			FillDetails: []TradeFillDetail{
				{
					FilledAccount: candidate.Maker.SubaccountID,
					AmountFilled:  amount,
					Price:         price,
					Fee:           "0",
				},
			},
			ManagerData: defaultManagerData(managerData),
		},
	}, nil
}

func defaultManagerData(managerData string) string {
	trimmed := strings.TrimSpace(managerData)
	if trimmed == "" {
		return "0x"
	}
	return trimmed
}

func normalizeAction(order orders.Order) (json.RawMessage, error) {
	raw := order.ActionJSON
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid json")
	}

	var action map[string]any
	if err := json.Unmarshal(raw, &action); err != nil {
		return nil, err
	}

	required := []string{"subaccount_id", "nonce", "module", "data", "expiry", "owner", "signer"}
	for _, field := range required {
		if _, ok := action[field]; !ok {
			return nil, fmt.Errorf("missing %s", field)
		}
	}

	if actionSubaccount, ok := action["subaccount_id"].(string); !ok || actionSubaccount != order.SubaccountID {
		return nil, fmt.Errorf("subaccount_id mismatch")
	}
	if actionNonce, ok := action["nonce"].(string); !ok || actionNonce != order.Nonce {
		return nil, fmt.Errorf("nonce mismatch")
	}
	if actionOwner, ok := action["owner"].(string); !ok || strings.ToLower(actionOwner) != strings.ToLower(order.OwnerAddress) {
		return nil, fmt.Errorf("owner mismatch")
	}
	if actionSigner, ok := action["signer"].(string); !ok || strings.ToLower(actionSigner) != strings.ToLower(order.SignerAddress) {
		return nil, fmt.Errorf("signer mismatch")
	}

	normalized, err := json.Marshal(action)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func extractModuleAddress(raw json.RawMessage) string {
	var action struct {
		Module string `json:"module"`
	}
	if err := json.Unmarshal(raw, &action); err != nil {
		return ""
	}
	return strings.ToLower(action.Module)
}
