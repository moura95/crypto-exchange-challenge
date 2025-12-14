package server

import (
	"encoding/json"
	"net/http"
	"time"

	v1 "github.com/moura95/crypto-exchange-challenge/api/v1"
	"github.com/moura95/crypto-exchange-challenge/config"
	"github.com/moura95/crypto-exchange-challenge/internal/engine"
	"github.com/moura95/crypto-exchange-challenge/internal/handler"
	"github.com/moura95/crypto-exchange-challenge/pkg/logger"
)

const Version = "1.0.0"

type Server struct {
	config           *config.Config
	engine           *engine.Engine
	orderHandler     *handler.OrderHandler
	accountHandler   *handler.AccountHandler
	orderbookHandler *handler.OrderbookHandler
	startTime        time.Time
}

func NewServer(cfg *config.Config) (*Server, error) {
	logger.Info("Initializing server...")

	// Initialize engine
	eng := engine.NewEngine()

	// Initialize handlers
	orderHandler := handler.NewOrderHandler(eng)
	accountHandler := handler.NewAccountHandler(eng.GetAccountManager())
	orderbookHandler := handler.NewOrderbookHandler(eng)

	return &Server{
		config:           cfg,
		engine:           eng,
		orderHandler:     orderHandler,
		accountHandler:   accountHandler,
		orderbookHandler: orderbookHandler,
		startTime:        time.Now(),
	}, nil
}

func (s *Server) Start() error {
	s.registerRoutes()

	logger.Infof("Server starting on %s (version %s)", s.config.HTTPServerAddress, Version)
	return http.ListenAndServe(s.config.HTTPServerAddress, nil)
}

func (s *Server) registerRoutes() {
	// Health check
	http.HandleFunc("/health", s.handleHealth)

	// Account routes
	http.HandleFunc("/api/v1/accounts/credit", s.accountHandler.Credit)
	http.HandleFunc("/api/v1/accounts/debit", s.accountHandler.Debit)
	http.HandleFunc("/api/v1/accounts/balance", s.accountHandler.GetBalance)

	// Order routes
	http.HandleFunc("/api/v1/orders", s.orderHandler.PlaceOrder)
	http.HandleFunc("/api/v1/orders/cancel", s.orderHandler.CancelOrder)

	// Orderbook routes
	http.HandleFunc("/api/v1/orderbook", s.orderbookHandler.GetOrderbook)

	logger.Info("Routes registered:")
	logger.Info("  GET  /health")
	logger.Info("  POST /api/v1/accounts/credit")
	logger.Info("  POST /api/v1/accounts/debit")
	logger.Info("  GET  /api/v1/accounts/balance?user_id={id}")
	logger.Info("  POST /api/v1/orders")
	logger.Info("  POST /api/v1/orders/cancel")
	logger.Info("  GET  /api/v1/orderbook?pair={pair}")
}

// handleHealth godoc
// @Summary Health check
// @Description Returns the health status of the API
// @Tags Health
// @Produce json
// @Success 200 {object} v1.HealthResponse "Service is healthy"
// @Router /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := v1.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Error encoding health response: %v", err)
	}
}
