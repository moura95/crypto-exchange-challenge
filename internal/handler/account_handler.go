package handler

import (
	"encoding/json"
	"net/http"
	"time"

	v1 "github.com/moura95/crypto-exchange-challenge/api/v1"
	"github.com/moura95/crypto-exchange-challenge/internal/account"
	"github.com/moura95/crypto-exchange-challenge/pkg/logger"
)

type AccountHandler struct {
	manager *account.Manager
}

func NewAccountHandler(manager *account.Manager) *AccountHandler {
	return &AccountHandler{
		manager: manager,
	}
}

// Credit godoc
// @Summary Credit asset to account
// @Description Add balance to a user's account
// @Tags Accounts
// @Accept json
// @Produce json
// @Param request body v1.CreditDebitRequest true "Credit details (includes user_id)"
// @Success 200 {object} v1.BalanceResponse "Credit successful"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Router /api/v1/accounts/credit [post]
func (h *AccountHandler) Credit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req v1.CreditDebitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		logger.Warningf("Credit - invalid JSON - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Validate
	if req.UserID == "" {
		h.sendError(w, "user_id is required", http.StatusBadRequest)
		logger.Warningf("Credit - missing user_id - Duration: %v", time.Since(start))
		return
	}
	if req.Asset == "" {
		h.sendError(w, "asset is required", http.StatusBadRequest)
		logger.Warningf("Credit - missing asset - Duration: %v", time.Since(start))
		return
	}
	if req.Amount <= 0 {
		h.sendError(w, "amount must be greater than 0", http.StatusBadRequest)
		logger.Warningf("Credit - invalid amount - Duration: %v", time.Since(start))
		return
	}

	// Credit
	if err := h.manager.Credit(req.UserID, req.Asset, req.Amount); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Credit failed - User: %s - Asset: %s - Amount: %.8f - Duration: %v - Error: %v",
			req.UserID, req.Asset, req.Amount, time.Since(start), err)
		return
	}

	// Get updated balance
	response := h.getBalanceResponse(req.UserID)
	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Credit success - User: %s - Asset: %s - Amount: %.8f - Status: 200 - Duration: %v",
		req.UserID, req.Asset, req.Amount, time.Since(start))
}

// Debit godoc
// @Summary Debit asset from account
// @Description Remove balance from a user's account
// @Tags Accounts
// @Accept json
// @Produce json
// @Param request body v1.CreditDebitRequest true "Debit details (includes user_id)"
// @Success 200 {object} v1.BalanceResponse "Debit successful"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Router /api/v1/accounts/debit [post]
func (h *AccountHandler) Debit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req v1.CreditDebitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		logger.Warningf("Debit - invalid JSON - Duration: %v - Error: %v", time.Since(start), err)
		return
	}

	// Validate
	if req.UserID == "" {
		h.sendError(w, "user_id is required", http.StatusBadRequest)
		logger.Warningf("Debit - missing user_id - Duration: %v", time.Since(start))
		return
	}
	if req.Asset == "" {
		h.sendError(w, "asset is required", http.StatusBadRequest)
		logger.Warningf("Debit - missing asset - Duration: %v", time.Since(start))
		return
	}
	if req.Amount <= 0 {
		h.sendError(w, "amount must be greater than 0", http.StatusBadRequest)
		logger.Warningf("Debit - invalid amount - Duration: %v", time.Since(start))
		return
	}

	// Debit
	if err := h.manager.Debit(req.UserID, req.Asset, req.Amount); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		logger.Warningf("Debit failed - User: %s - Asset: %s - Amount: %.8f - Duration: %v - Error: %v",
			req.UserID, req.Asset, req.Amount, time.Since(start), err)
		return
	}

	// Get updated balance
	response := h.getBalanceResponse(req.UserID)
	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Debit success - User: %s - Asset: %s - Amount: %.8f - Status: 200 - Duration: %v",
		req.UserID, req.Asset, req.Amount, time.Since(start))
}

// GetBalance godoc
// @Summary Get account balance
// @Description Get all balances for a user's account
// @Tags Accounts
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} v1.BalanceResponse "Balance retrieved successfully"
// @Failure 400 {object} v1.ErrorResponse "Invalid request"
// @Router /api/v1/accounts/balance [get]
func (h *AccountHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.sendError(w, "user_id query parameter is required", http.StatusBadRequest)
		logger.Warningf("Get balance - missing user_id - Duration: %v", time.Since(start))
		return
	}

	response := h.getBalanceResponse(userID)
	h.sendJSON(w, response, http.StatusOK)

	logger.Infof("Get balance success - User: %s - Assets: %d - Status: 200 - Duration: %v",
		userID, len(response.Balances), time.Since(start))
}

// Helper methods

func (h *AccountHandler) getBalanceResponse(userID string) v1.BalanceResponse {
	balances := h.manager.GetAllBalances(userID)

	items := make([]v1.BalanceItem, 0, len(balances))
	for asset, balance := range balances {
		items = append(items, v1.BalanceItem{
			Asset:     asset,
			Available: balance.Available,
			Locked:    balance.Locked,
			Total:     balance.Total(),
		})
	}

	return v1.BalanceResponse{
		UserID:   userID,
		Balances: items,
	}
}

func (h *AccountHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Errorf("Error encoding JSON response: %v", err)
	}
}

func (h *AccountHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	h.sendJSON(w, v1.ErrorResponse{Error: message}, statusCode)
}
