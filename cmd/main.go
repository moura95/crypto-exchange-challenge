package main

import (
	"log"

	"github.com/moura95/crypto-exchange-challenge/config"
	server "github.com/moura95/crypto-exchange-challenge/internal"

	_ "github.com/moura95/crypto-exchange-challenge/docs" // Importar docs gerados
)

// @title Crypto Exchange API
// @version 1.0.0
// @description Central Limit Order Book (CLOB) with matching engine for trading crypto assets
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
