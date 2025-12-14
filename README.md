# Go Exchange

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org)

**Central Limit Order Book (CLOB) with Matching Engine** - Complete order book system for crypto asset trading with automatic order execution.

## 📋 About the Project

Implementation of a simplified **Central Limit Order Book (CLOB)** with matching engine to execute limit and market orders. The system manages user accounts, asset balances, and automatically executes trades when buy and sell orders cross in the book.

This project was developed as a response to a technical challenge, focusing on:
- ✨ Clean and well-structured code
- 🎯 Scalable and maintainable architecture
- 🧪 Comprehensive testing (unit, integration)
- 📊 Performance and thread-safety

---

## ✨ Implemented Features

### ✅ Required Features
- **Place Order** - Create limit order in the orderbook
- **Cancel Order** - Cancel existing order

### 🎁 Bonus Features
- **Credit** - Add balance to an account
- **Debit** - Remove balance from an account
- **Get Balance** - Query all user balances
- **Get Orderbook** - View current book state (bids/asks)

### 🌟 Additional Features Implemented
- **Market Orders** - Market orders with immediate execution
- **Partial Fill** - Support for partial order execution
- **Price-Time Priority (FIFO)** - Matching by best price + chronological order
- **Self-Trade Prevention** - Prevents users from trading against themselves
- **Balance Locking** - Automatic balance reservation when creating orders
- **Price Improvement** - Returns difference when executing at better price
- **Concurrent Safe** - Thread-safety with mutexes (RWMutex)
- **Trade History** - Complete trade execution history
- **Swagger Documentation** - Interactive API docs
- **Comprehensive Tests** - Test coverage with edge cases

---

## 💡 Usage Examples

### ⚠️ Important Note about User ID
This system **does not have authentication/user control**. You can use any `user_id` in requests (example: "1", "alice", "bob", etc.). The system only manages balances and orders by user_id, but does not validate if the user exists or is authenticated.

### Credit Balance

Add funds to a user account:

```json
POST /api/v1/accounts/credit

{
  "user_id": "1",
  "asset": "BTC",
  "amount": 5
}
```

```json
POST /api/v1/accounts/credit

{
  "user_id": "1",
  "asset": "BRL",
  "amount": 500000
}
```

### Place Limit Order

Create an order with a specific price:

```json
POST /api/v1/orders

{
  "user_id": "1",
  "pair": "BTC/BRL",
  "side": "ask",
  "type": "limit",
  "price": 51000,
  "amount": 1
}
```

**Valid options:**
- `side`: `"bid"` (buy) or `"ask"` (sell)
- `type`: `"limit"` (specific price) or `"market"` (immediate execution at best price)

### Place Market Order

Execute immediately at the best available price:

```json
POST /api/v1/orders

{
  "user_id": "2",
  "pair": "BTC/BRL",
  "side": "bid",
  "type": "market",
  "amount": 0.5
}
```

### Cancel Order

Cancel an existing order:

```json
POST /api/v1/orders/cancel

{
  "user_id": "1",
  "order_id": "1"
}
```

### Check Balance

Query all balances for a user:

```http
GET /api/v1/accounts/balance?user_id=1
```

Response:
```json
{
  "user_id": "1",
  "balances": {
    "BTC": {
      "available": 4.0,
      "locked": 1.0,
      "total": 5.0
    },
    "BRL": {
      "available": 449000,
      "locked": 51000,
      "total": 500000
    }
  }
}
```

### View Orderbook

See current state of the orderbook:

```http
GET /api/v1/orderbook?pair=BTC/BRL
```

Response:
```json
{
  "pair": "BTC/BRL",
  "bids": [
    {
      "price": 50000,
      "amount": 2.5,
      "total": 125000
    }
  ],
  "asks": [
    {
      "price": 51000,
      "amount": 1.0,
      "total": 51000
    }
  ]
}
```

---

## 🏗️ Architecture

The system follows **Clean Architecture** principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Layer                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                         │
│  │  Order   │ │ Account  │ │Orderbook │                         │
│  │ Handler  │ │ Handler  │ │ Handler  │                         │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘                         │
└───────┼────────────┼────────────┼──────────────────────────────┘
        │            │            │
        ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      MATCHING ENGINE                             │
│                                                                  │
│   PlaceOrder()      → Validate + Lock + Match + Transfer        │
│   PlaceMarketOrder()→ Estimate cost + Lock + Match + Refund     │
│   CancelOrder()     → Validate ownership + Remove + Unlock      │
│   GetOrderbook()    → Thread-safe book snapshot                 │
│                                                                  │
└───────────┬─────────────────────────────────┬───────────────────┘
            │                                 │
            ▼                                 ▼
┌───────────────────────┐       ┌───────────────────────┐
│    ACCOUNT MANAGER    │       │      ORDERBOOK        │
│                       │       │                       │
│  Accounts:            │       │  Bids (sorted desc):  │
│   map[userID]         │       │   []*Limit            │
│    map[asset]         │       │                       │
│     *Balance          │       │  Asks (sorted asc):   │
│                       │       │   []*Limit            │
│  Methods:             │       │                       │
│   • Credit()          │       │  Limit:               │
│   • Debit()           │       │   • PriceTicks (int64)│
│   • Lock()            │       │   • Orders[] (FIFO)   │
│   • Unlock()          │       │   • TotalVolume       │
│   • DebitLocked()     │       │                       │
│                       │       │  Orders map by ID     │
└───────────────────────┘       └───────────────────────┘
```

### Main Components

**1. HTTP Layer (Handlers)**
- Request validation
- DTO conversion
- Structured logging
- Error handling

**2. Matching Engine**
- Orchestrates operations between orderbook and account manager
- Ensures operation atomicity
- Manages balance locks
- Executes transfers after matches

**3. Orderbook**
- Maintains sorted bids (buy) and asks (sell)
- FIFO matching within same price level
- Self-trade prevention
- Thread-safe with RWMutex

**4. Account Manager**
- Manages balances for multiple assets per user
- Supports Available/Locked balance
- Prevents double-spending
- Thread-safe

---

## 🛠️ Technologies

- **Go 1.24** - Main language
- **Standard Library HTTP** - Native server (as per challenge requirements)
- **In-Memory Storage** - Thread-safe maps with sync.RWMutex
- **Swagger/OpenAPI** - API documentation
- **Docker** - Containerization

### External Dependencies (tooling only)
```go
require (
    github.com/swaggo/http-swagger v1.3.4  // Swagger UI
    github.com/swaggo/swag v1.16.6         // Swagger generator
)
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.24+** ([Download](https://golang.org/dl/))
- **Make** (optional, but recommended)
- **Docker** (optional, for containerized execution)

### Installation and Execution

```bash
# 1. Clone the repository
git clone https://github.com/moura95/crypto-exchange-challenge
cd crypto-exchange-challenge

# 2. Configure environment variables
cp .envexample .env
# Edit .env if needed (default: HTTP_SERVER_ADDRESS=0.0.0.0:8080)

# 3. Install dependencies
go mod download

# 4. Run the server
make run
# Or without Make:
# go run ./cmd

# 5. Access the API
# Health check: http://localhost:8080/health
# Swagger UI:   http://localhost:8080/swagger/index.html

# 6. Download Postman Collection
# docs/CryptoExchange.postman_collection.json
```

### With Docker

```bash
# Build and run with docker-compose
make docker-run

# Stop containers
make docker-stop

# Complete cleanup
make docker-clean
```

---

## 📚 API Endpoints

### Health Check
```http
GET /health
```

### Account Management
```http
POST /api/v1/accounts/credit              # Add balance
POST /api/v1/accounts/debit               # Remove balance
GET  /api/v1/accounts/balance?user_id={id} # Query balances
```

### Order Management
```http
POST /api/v1/orders                       # Create order (limit or market)
POST /api/v1/orders/cancel                # Cancel order
```

### Orderbook
```http
GET /api/v1/orderbook?pair={pair}         # View orderbook (e.g., BTC/BRL)
```

### 📖 Interactive Documentation

Access **Swagger UI** at: `http://localhost:8080/swagger/index.html`

All endpoints are fully documented with request/response examples, parameter descriptions, and can be tested directly from the browser.

---

## 🧪 Testing

The project has comprehensive test coverage with multiple test types:

```bash
# Run all tests
make test
```

### Types of Tests Implemented

1. **Unit Tests** - Test isolated components
2. **Integration Tests** - Test complete flows
3. **Edge Case Tests** - Test extreme scenarios:
   - Insufficient balance
   - Self-trade prevention
   - Partial fills
   - Price improvement
   - Double cancellation
   - Market order with insufficient liquidity

---

## 🧠 Technical Decisions

### 1. In-Memory Storage

**Decision:** Use in-memory maps with `sync.RWMutex`

**Rationale:**
- ✅ Simplicity: No database dependencies
- ✅ Performance: O(1) lookup for critical operations
- ✅ Thread-safety: RWMutex allows multiple simultaneous reads
- ✅ Appropriate for scope: Challenge doesn't require persistence

**Trade-off:** Data doesn't persist between restarts

**In Production:** Would use Redis for cache + PostgreSQL for persistence

---

### 2. Standard Library HTTP

**Decision:** Use native `net/http` instead of frameworks (Gin, Echo, Fiber)

**Rationale:**
- ✅ Simplicity: Basic REST API doesn't need a framework
- ✅ Zero overhead: Maximum performance
- ✅ Facilitates analysis: More straightforward code

**In Production:** Would consider Gin/Echo for features like:
- Middleware chains
- Request validation
- Auto-binding

---

### 3. Price-Time Priority (FIFO)

**Decision:** Matching by best price + arrival order

**Implementation:**
```go
// Bids sorted by price DESC (highest first)
sort.Slice(ob.bids, func(i, j int) bool {
    return ob.bids[i].PriceTicks > ob.bids[j].PriceTicks
})

// Asks sorted by price ASC (lowest first)  
sort.Slice(ob.asks, func(i, j int) bool {
    return ob.asks[i].PriceTicks < ob.asks[j].PriceTicks
})

// Within each Limit: FIFO (slice preserves insertion order)
```

**Why FIFO?**
- ✅ Fair: First to arrive has priority
- ✅ Market standard: Used by most exchanges
- ✅ Prevents front-running at same price

---

### 4. Balance Locking

**Decision:** Reserve balance when creating order

**Flow:**
1. **PlaceOrder** → Lock funds (quote for buy, base for sell)
2. **Match** → DebitLocked + Credit counterparty
3. **Cancel** → Unlock remaining
4. **Partial Fill** → Refund price improvement

**Benefits:**
- ✅ Prevents double-spending
- ✅ Ensures liquidity during active order
- ✅ Atomicity: Lock → Match → Transfer

**Example:**
```go
// User wants to BUY 1 BTC @ 50k BRL
Lock("user", "BRL", 50000)     // Reserve quote

// Match at 49k (price improvement!)
DebitLocked("user", "BRL", 49000)
Unlock("user", "BRL", 1000)    // Refund difference
Credit("user", "BTC", 1.0)
```

---

### 5. Monetary Precision

**Decision:** Use `float64` with **tick normalization** system

**Current Implementation:**
```go
// Bidirectional conversion to int64 (avoids rounding errors)
func PriceToTicks(price, tick float64) int64 {
    return int64(math.Round(price / tick))
}

func TicksToPrice(ticks int64, tick float64) float64 {
    return float64(ticks) * tick
}

// Sorting and comparison always in int64
type Limit struct {
    PriceTicks  int64  // Source of truth
    Orders      []*Order
    TotalVolume float64
}
```

**Tick Sizes:**
- Prices: `0.01 BRL` (1 cent)
- Quantities: `0.00000001 BTC` (1 satoshi)

**Why float64?**
- ✅ Readability simplicity (legible business logic)
- ✅ No external dependencies (shopspring/decimal)
- ✅ Tick system solves critical precision problems
- ✅ Appropriate for challenge scope

**Known Limitations:**
- ⚠️ Float64 may have imprecisions in complex arithmetic operations
- ⚠️ Not recommended for production with real financial values

**If I had more time / For Production:**
```go
import "github.com/shopspring/decimal"

type Order struct {
    Price  decimal.Decimal  // instead of float64
    Amount decimal.Decimal
}

// Benefits:
// ✅ Arbitrary precision
// ✅ Financial compliance
// ✅ Complete elimination of rounding errors
```

**Justification for Choice:**
- ✅ Tick system solves the problem for challenge scope
- ✅ Float64 keeps code readable without external deps
- ✅ Architecture ready for migration (just change types)

**References:**
- [shopspring/decimal - Go package](https://github.com/shopspring/decimal)

---

### 6. Concurrency Model

**Decision:** RWMutex in critical components

**Granularity:**
```go
type Engine struct {
    orderbooks map[string]*Orderbook
    accounts   *AccountManager
    mu         sync.RWMutex  // 1 mutex for entire engine
}

type Orderbook struct {
    bids []*Limit
    asks []*Limit
    mu   sync.RWMutex  // 1 mutex per orderbook
}

type AccountManager struct {
    accounts map[string]map[string]*Balance
    mu       sync.RWMutex  // 1 mutex for all accounts
}
```

**Why RWMutex?**
- ✅ Allows multiple simultaneous reads (GetBalance, GetOrderbook)
- ✅ Exclusivity only on writes (PlaceOrder, Credit)
- ✅ Better performance than simple Mutex

---

### 7. Error Handling

**Decision:** Return explicit errors, no panic

**Examples:**
```go
// ✅ Good: Return error
if balance.Available < amount {
    return ErrInsufficientBalance
}

// ❌ Bad: Panic
if balance.Available < amount {
    panic("insufficient balance")
}
```

**Advantages:**
- ✅ Caller decides how to handle
- ✅ Graceful degradation
- ✅ Facilitates testing

---

## 📁 Project Structure

```
crypto-exchange-challenge/
├── cmd/
│   └── main.go                    # Entry point + Swagger annotations
│
├── config/
│   └── config.go                  # Environment config
│
├── api/v1/                        # DTOs (Request/Response)
│   ├── account.go
│   ├── order.go
│   ├── orderbook.go
│   ├── error.go
│   └── health.go
│
├── internal/
│   ├── server.go                  # HTTP server setup
│   │
│   ├── handler/                   # HTTP handlers
│   │   ├── account_handler.go
│   │   ├── order_handler.go
│   │   └── orderbook_handler.go
│   │
│   ├── engine/                    # Matching engine (orchestrator)
│   │   ├── engine.go
│   │   ├── engine_test.go
│   │   ├── types.go
│   │   ├── errors.go
│   │   └── testing_helpers.go
│   │
│   ├── orderbook/                 # Order book logic
│   │   ├── orderbook.go
│   │   ├── orderbook_test.go
│   │   ├── orderbook_market_test.go
│   │   ├── limit.go
│   │   ├── limit_test.go
│   │   ├── order.go
│   │   ├── order_test.go
│   │   ├── matche.go
│   │   ├── types.go
│   │   ├── errors.go
│   │   └── testing_helpers.go
│   │
│   └── account/                   # Account management
│       ├── manager.go
│       ├── manager_test.go
│       ├── balance.go
│       ├── errors.go
│       └── testing_helpers.go
│
├── pkg/
│   ├── logger/                    # Structured logging
│   │   ├── logger.go
│   │   └── logger_test.go
│   │
│   └── utils/                     # Utilities
│       └── tick.go                # Tick normalization
│
├── docs/                          # Swagger docs (auto-generated) / Postman Collection
│   ├── CryptoExchange.postman_collection.json
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── .env.example                   # Environment variables template
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

### Organization Patterns

**Naming Convention:**
- `internal/` - Private code (cannot be imported by other projects)
- `pkg/` - Reusable code (can be imported)
- `api/v1/` - Versioned DTOs

**Test Files:**
- `*_test.go` - Co-located with source code
- `testing_helpers.go` - Reusable assertions and setup

---

## 🎯 Main Flows

### PlaceOrder (Limit)

```
1. Validate Request (userID, pair, side, price, amount)
2. Create Order object
3. Determine asset to lock (quote for buy, base for sell)
4. Lock funds in AccountManager
5. Place order in Orderbook
6. Matching loop:
   - Find best counterparty orders
   - Fill order incrementally
   - Execute balance transfers for each match
   - Remove filled orders from book
7. Refund price improvement (if buy matched better)
8. Return order + matches
```

### PlaceOrder (Market)

```
1. Validate Request
2. Create Market Order
3. Estimate cost by scanning orderbook
4. Lock estimated cost
5. Execute market order (IOC - Immediate or Cancel)
6. Transfer balances for each match
7. Refund unused locked amount
8. Return order + matches
```

### CancelOrder

```
1. Validate ownership (order.UserID == requester)
2. Remove order from orderbook
3. Unlock remaining balance
4. Mark order as cancelled
5. Return cancelled order
```

---

## 🔍 Key Concepts

### Order States

```go
const (
    OrderOpen            = "open"              // In book, not matched
    OrderPartiallyFilled = "partially_filled"  // Some matched, rest in book
    OrderFilled          = "filled"            // Fully matched
    OrderCancelled       = "cancelled"         // Removed by user
)
```

### Order Types & Sides

```go
// Order Types
const (
    OrderTypeLimit  = "limit"   // Order with specific price
    OrderTypeMarket = "market"  // Immediate execution at best available price
)

// Order Sides
const (
    SideBid = "bid"  // BUY order
    SideAsk = "ask"  // SELL order
)
```

**Examples:**
- `"side": "bid"` + `"type": "limit"` → Buy BTC at a specific maximum price
- `"side": "ask"` + `"type": "limit"` → Sell BTC at a specific minimum price
- `"side": "bid"` + `"type": "market"` → Buy BTC immediately at best available price
- `"side": "ask"` + `"type": "market"` → Sell BTC immediately at best available price

### Balance Structure

```go
type Balance struct {
    Available float64  // Free to use
    Locked    float64  // Reserved for active orders
}

func (b *Balance) Total() float64 {
    return b.Available + b.Locked
}
```

---

## 🚧 Future Improvements

If I had more time, I would implement:

### 1. Persistence
- [ ] PostgreSQL for orderbook + trades
- [ ] Redis for orderbook snapshot cache

### 2. Numerical Precision
- [ ] Migrate to `decimal.Decimal` (shopspring/decimal)
- [ ] Support different tick sizes per pair

### 3. Advanced Features
- [ ] WebSocket for real-time orderbook updates
- [ ] Additional order types (Stop-Loss, Stop-Limit, FOK, IOC)
- [ ] Optimized multi-pair support
- [ ] Rate limiting per user

### 4. Operations
- [ ] Detailed health checks (DB, memory, goroutines)
- [ ] Graceful shutdown
- [ ] Metrics and monitoring

### 5. Security & Authentication
- [ ] **User authentication/authorization system**
   - JWT/OAuth2 implementation
   - User registration and login
   - Validate user_id against authenticated sessions
- [ ] Rate limiting per user
- [ ] Input sanitization
- [ ] HTTPS/TLS
- [ ] API key management

---
## 👨‍💻 Author

**Guilherme Moura**  
*Software Engineer*

- 🐙 GitHub: [@moura95](https://github.com/moura95)
- 💼 LinkedIn: [Guilherme Moura](https://linkedin.com/in/guilherme-moura95)