package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/instruments"
	oraclemodule "github.com/numofx/matching-backend/internal/oracles/btcvar30"
	"github.com/numofx/matching-backend/internal/orders"
)

type oracleReader interface {
	Latest() (oraclemodule.Payload, bool)
	History(ctx context.Context, limit int) ([]oraclemodule.Payload, error)
}

type Server struct {
	cfg         config.Config
	pool        *pgxpool.Pool
	orders      *orders.Repository
	instruments *instruments.Registry
	oracle      oracleReader
}

func NewServer(cfg config.Config, pool *pgxpool.Pool, registry *instruments.Registry, oracle oracleReader) *Server {
	return &Server{
		cfg:         cfg,
		pool:        pool,
		orders:      orders.NewRepository(pool),
		instruments: registry,
		oracle:      oracle,
	}
}

func (s *Server) Run() error {
	router := chi.NewRouter()
	router.Get("/healthz", s.handleHealth)
	router.Get("/v1/book", s.handleBook)
	router.Post("/v1/orders", s.handleCreateOrder)
	router.Post("/v1/orders/cancel", s.handleCancelOrder)
	router.Get("/oracle/btcvar30/latest", s.handleBTCVar30Latest)
	router.Get("/oracle/btcvar30/history", s.handleBTCVar30History)

	slog.Info("api listening", "addr", s.cfg.APIAddr)
	return http.ListenAndServe(s.cfg.APIAddr, router)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	market := s.resolveMarket(r)
	if market.AssetAddress == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown market"})
		return
	}

	bids, asks, err := s.orders.ListBook(r.Context(), strings.ToLower(market.AssetAddress), market.SubID, 25)
	if err != nil {
		slog.Error("list book", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load book"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"market":        market.Symbol,
		"asset_address": strings.ToLower(market.AssetAddress),
		"sub_id":        market.SubID,
		"bids":          bids,
		"asks":          asks,
	})
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	params, err := req.toParams(s.cfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	order, err := s.orders.Create(r.Context(), params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "duplicate order") {
			statusCode = http.StatusConflict
		}
		slog.Error("create order", "error", err)
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"order": order})
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	var req cancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if err := req.validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	order, err := s.orders.CancelByOwnerNonce(r.Context(), orders.CancelOrderParams{
		OwnerAddress: strings.ToLower(req.OwnerAddress),
		Nonce:        req.Nonce,
	})
	if err != nil {
		if errors.Is(err, orders.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "active order not found"})
			return
		}
		slog.Error("cancel order", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to cancel order"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

func (s *Server) handleBTCVar30Latest(w http.ResponseWriter, _ *http.Request) {
	if s.oracle == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "btcvar30 oracle is not configured"})
		return
	}

	payload, ok := s.oracle.Latest()
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "btcvar30 oracle has no data"})
		return
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleBTCVar30History(w http.ResponseWriter, r *http.Request) {
	if s.oracle == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "btcvar30 oracle is not configured"})
		return
	}

	limit := 100
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 1000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 1000"})
			return
		}
		limit = parsed
	}

	items, err := s.oracle.History(r.Context(), limit)
	if err != nil {
		slog.Error("load btcvar30 history", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load btcvar30 history"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"symbol":  oraclemodule.Symbol,
		"history": items,
	})
}

func (s *Server) resolveMarket(r *http.Request) instruments.Metadata {
	if s.instruments == nil {
		return instruments.Metadata{}
	}

	if symbol := strings.TrimSpace(r.URL.Query().Get("symbol")); symbol != "" {
		if item, ok := s.instruments.BySymbol(symbol); ok {
			return item
		}
	}

	if assetAddress := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("asset_address"))); assetAddress != "" {
		if item, ok := s.instruments.ByAssetAddress(assetAddress); ok {
			return item
		}
	}

	if item, ok := s.instruments.BySymbol(instruments.BTCConvexPerpSymbol); ok {
		return item
	}
	return instruments.Metadata{}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
