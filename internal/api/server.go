package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/orders"
)

type Server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	orders *orders.Repository
}

func NewServer(cfg config.Config, pool *pgxpool.Pool) *Server {
	return &Server{
		cfg:    cfg,
		pool:   pool,
		orders: orders.NewRepository(pool),
	}
}

func (s *Server) Run() error {
	router := chi.NewRouter()
	router.Get("/healthz", s.handleHealth)
	router.Get("/v1/book", s.handleBook)
	router.Post("/v1/orders", s.handleCreateOrder)
	router.Post("/v1/orders/cancel", s.handleCancelOrder)

	slog.Info("api listening", "addr", s.cfg.APIAddr)
	return http.ListenAndServe(s.cfg.APIAddr, router)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	bids, asks, err := s.orders.ListBook(r.Context(), strings.ToLower(s.cfg.BTCPerpAssetAddress), "0", 25)
	if err != nil {
		slog.Error("list book", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load book"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"market":        "BTC-PERP",
		"asset_address": strings.ToLower(s.cfg.BTCPerpAssetAddress),
		"sub_id":        "0",
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

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
