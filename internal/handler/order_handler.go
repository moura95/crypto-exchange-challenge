package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	v1 "github.com/moura95/crypto-exchange-challenge/api/v1"
	"github.com/moura95/crypto-exchange-challenge/internal/engine"
	"github.com/moura95/crypto-exchange-challenge/internal/orderbook"
	"github.com/moura95/crypto-exchange-challenge/pkg/logger"
)

type OrderHandler struct {
	engine *engine.Engine
}

func NewOrderHandler(engine *engine.Engine) *OrderHandler {
	return &OrderHandler{
		engine: engine,
	}
}

// PlaceOrder godoc
// @Summary Place a new order
// @Description Create a limit or market order
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body v1.PlaceOrderRequest true "Order details"
// @Success 200 {object} v1.PlaceOrderResponse "Order placed successfully"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Failure 500 {object} v1.ErrorResponse "Internal server error"
// @Router /api/v1/orders [post]
func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req v1.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		logger.Warningf("Place order - invalid JSON - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Validação
	if err := h.validatePlaceOrderRequest(req); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Place order - validation failed - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Parse pair (BTC/BRL -> Base: BTC, Quote: BRL)
	pair, err := h.parsePair(req.Pair)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Place order - invalid pair - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Parse side
	side, err := h.parseSide(req.Side)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Place order - invalid side - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	var order *orderbook.Order
	var matches []orderbook.Match

	// Place order based on type
	if req.Type == "market" {
		order, matches, err = h.engine.PlaceMarketOrder(req.UserID, pair, side, req.Amount)
	} else {
		order, matches, err = h.engine.PlaceOrder(req.UserID, pair, side, req.Price, req.Amount)
	}

	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Place order failed - User: %s - Pair: %s - Duration: %v - Error: %v",
			req.UserID, req.Pair, time.Since(start), err)
		return
	}

	// Convert to response
	response := v1.PlaceOrderResponse{
		Order:   h.orderToResponse(order, req.Pair),
		Matches: h.matchesToResponse(matches),
	}

	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Place order success - User: %s - Pair: %s - Type: %s - Side: %s - Price: %.2f - Amount: %.8f - Matches: %d - Status: 200 - Duration: %v",
		req.UserID, req.Pair, req.Type, req.Side, req.Price, req.Amount, len(matches), time.Since(start))
}

// CancelOrder godoc
// @Summary Cancel an order
// @Description Cancel an existing order by ID
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body v1.CancelOrderRequest true "Cancel order details"
// @Success 200 {object} v1.OrderResponse "Order cancelled successfully"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Failure 404 {object} v1.ErrorResponse "Order not found"
// @Router /api/v1/orders/cancel [post]
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req v1.CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		logger.Warningf("Cancel order - invalid JSON - Duration: %v", time.Since(start))
		return
	}

	// Validate
	if req.UserID == "" {
		h.sendError(w, "user_id is required", http.StatusBadRequest)
		logger.Warningf("Cancel order - missing user_id - Duration: %v", time.Since(start))
		return
	}
	if req.Pair == "" {
		h.sendError(w, "pair is required", http.StatusBadRequest)
		logger.Warningf("Cancel order - missing pair - Duration: %v", time.Since(start))
		return
	}
	if req.OrderID <= 0 {
		h.sendError(w, "order_id must be greater than 0", http.StatusBadRequest)
		logger.Warningf("Cancel order - invalid order_id - Duration: %v", time.Since(start))
		return
	}

	// Parse pair
	pair, err := h.parsePair(req.Pair)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Cancel order - invalid pair - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Cancel order
	cancelledOrder, err := h.engine.CancelOrder(req.UserID, pair, req.OrderID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, engine.ErrOrderNotFound) {
			statusCode = http.StatusNotFound
		} else if errors.Is(err, engine.ErrUnauthorized) {
			statusCode = http.StatusUnauthorized
		}

		h.sendError(w, err.Error(), statusCode)
		logger.Warningf("Cancel order failed - User: %s - OrderID: %d - Duration: %v - Error: %v",
			req.UserID, req.OrderID, time.Since(start), err)
		return
	}

	response := h.orderToResponse(cancelledOrder, req.Pair)
	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Cancel order success - User: %s - OrderID: %d - Status: 200 - Duration: %v",
		req.UserID, req.OrderID, time.Since(start))
}

// Helper methods

func (h *OrderHandler) validatePlaceOrderRequest(req v1.PlaceOrderRequest) error {
	if req.UserID == "" {
		return errors.New("user_id is required")
	}
	if req.Pair == "" {
		return errors.New("pair is required")
	}
	if req.Side == "" {
		return errors.New("side is required")
	}
	if req.Type == "" {
		return errors.New("type is required")
	}
	if req.Type != "limit" && req.Type != "market" {
		return errors.New("type must be 'limit' or 'market'")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if req.Type == "limit" && req.Price <= 0 {
		return errors.New("price must be greater than 0 for limit orders")
	}
	return nil
}

func (h *OrderHandler) parsePair(pairStr string) (engine.Pair, error) {
	parts := strings.Split(pairStr, "/")
	if len(parts) != 2 {
		return engine.Pair{}, errors.New("pair must be in format BASE/QUOTE (e.g., BTC/BRL)")
	}

	pair := engine.Pair{
		Base:  strings.ToUpper(parts[0]),
		Quote: strings.ToUpper(parts[1]),
	}

	if !pair.IsValid() {
		return engine.Pair{}, errors.New("invalid pair: quote must be BRL")
	}

	return pair, nil
}

func (h *OrderHandler) parseSide(sideStr string) (orderbook.Side, error) {
	side := orderbook.Side(strings.ToLower(sideStr))
	if side != orderbook.Bid && side != orderbook.Ask {
		return "", errors.New("side must be 'bid' or 'ask'")
	}
	return side, nil
}

func (h *OrderHandler) orderToResponse(order *orderbook.Order, pairStr string) v1.OrderResponse {
	return v1.OrderResponse{
		ID:           order.ID,
		UserID:       order.UserID,
		Pair:         pairStr,
		Side:         string(order.Side),
		Type:         string(order.Type),
		Price:        order.Price,
		Amount:       order.Amount,
		FilledAmount: order.FilledAmount,
		State:        string(order.State),
		Timestamp:    order.Timestamp,
	}
}

func (h *OrderHandler) matchesToResponse(matches []orderbook.Match) []v1.MatchResponse {
	result := make([]v1.MatchResponse, len(matches))
	for i, m := range matches {
		result[i] = v1.MatchResponse{
			BidOrderID: m.Bid.ID,
			AskOrderID: m.Ask.ID,
			Price:      m.Price,
			SizeFilled: m.SizeFilled,
			Timestamp:  m.Timestamp,
		}
	}
	return result
}

func (h *OrderHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Errorf("Error encoding JSON response: %v", err)
	}
}

func (h *OrderHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	h.sendJSON(w, v1.ErrorResponse{Error: message}, statusCode)
}
