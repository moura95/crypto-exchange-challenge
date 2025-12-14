package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	v1 "github.com/moura95/crypto-exchange-challenge/api/v1"
	"github.com/moura95/crypto-exchange-challenge/internal/engine"
	"github.com/moura95/crypto-exchange-challenge/internal/orderbook"
	"github.com/moura95/crypto-exchange-challenge/pkg/logger"
)

type OrderbookHandler struct {
	engine *engine.Engine
}

func NewOrderbookHandler(engine *engine.Engine) *OrderbookHandler {
	return &OrderbookHandler{
		engine: engine,
	}
}

// GetOrderbook godoc
// @Summary Get orderbook
// @Description Get the current orderbook for a trading pair
// @Tags Orderbook
// @Produce json
// @Param pair query string true "Trading pair (e.g., BTC/BRL)"
// @Success 200 {object} v1.OrderbookResponse "Orderbook retrieved successfully"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Failure 404 {object} v1.ErrorResponse "Orderbook not found"
// @Router /api/v1/orderbook [get]
func (h *OrderbookHandler) GetOrderbook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	pairStr := r.URL.Query().Get("pair")
	if pairStr == "" {
		h.sendError(w, "pair query parameter is required (e.g., BTC/BRL)", http.StatusBadRequest)
		logger.Warningf("Get orderbook - missing pair - Duration: %v", time.Since(start))
		return
	}

	// Parse pair
	pair, err := h.parsePair(pairStr)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Get orderbook - invalid pair - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Get orderbook
	ob := h.engine.GetOrderbook(pair)
	if ob == nil {
		h.sendError(w, "Orderbook not found", http.StatusNotFound)
		logger.Infof("Get orderbook - not found - Pair: %s - Status: 404 - Duration: %v",
			pairStr, time.Since(start))
		return
	}

	// Convert to response
	response := h.orderbookToResponse(pair, ob)
	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Get orderbook success - Pair: %s - Bids: %d - Asks: %d - Status: 200 - Duration: %v",
		pairStr, len(response.Bids), len(response.Asks), time.Since(start))
}

// Helper methods

func (h *OrderbookHandler) parsePair(pairStr string) (engine.Pair, error) {
	parts := strings.Split(pairStr, "/")
	if len(parts) != 2 {
		return engine.Pair{}, &PairError{pairStr}
	}

	pair := engine.Pair{
		Base:  strings.ToUpper(parts[0]),
		Quote: strings.ToUpper(parts[1]),
	}

	if !pair.IsValid() {
		return engine.Pair{}, &PairError{pairStr}
	}

	return pair, nil
}

func (h *OrderbookHandler) orderbookToResponse(pair engine.Pair, ob *orderbook.Orderbook) v1.OrderbookResponse {
	bids := ob.Bids()
	asks := ob.Asks()

	bidLevels := make([]v1.LimitLevel, len(bids))
	for i, limit := range bids {
		bidLevels[i] = v1.LimitLevel{
			Price:       limit.Price(engine.PriceTick),
			TotalVolume: limit.TotalVolume,
		}
	}

	askLevels := make([]v1.LimitLevel, len(asks))
	for i, limit := range asks {
		askLevels[i] = v1.LimitLevel{
			Price:       limit.Price(engine.PriceTick),
			TotalVolume: limit.TotalVolume,
		}
	}

	return v1.OrderbookResponse{
		Pair:           pair.String(),
		Bids:           bidLevels,
		Asks:           askLevels,
		Spread:         ob.Spread(),
		BidTotalVolume: ob.BidTotalVolume(),
		AskTotalVolume: ob.AskTotalVolume(),
	}
}

func (h *OrderbookHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Errorf("Error encoding JSON response: %v", err)
	}
}

func (h *OrderbookHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	h.sendJSON(w, v1.ErrorResponse{Error: message}, statusCode)
}

// Custom error type
type PairError struct {
	Pair string
}

func (e *PairError) Error() string {
	return "invalid pair format: " + e.Pair + " (expected format: BASE/QUOTE, e.g., BTC/BRL)"
}
