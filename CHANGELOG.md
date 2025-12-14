# Changelog
## [1.0.0] - 2024-12-14

### Added

#### Core Business Logic
- **Orderbook Management**
    - Central Limit Order Book (CLOB) implementation with price-time priority
    - Support for limit orders with automatic matching
    - Support for market orders
    - Self-trade prevention mechanism
    - Partial fill support with state tracking
    - Price improvement refunds for buy orders
    - Thread-safe operations using RWMutex

- **Account Management**
    - Multi-asset balance tracking per user
    - Credit/Debit operations with validation
    - Balance locking mechanism for active orders
    - Unlock operations for order cancellation
    - DebitLocked for trade settlement
    - Thread-safe balance operations

- **Matching Engine**
    - Orchestration between orderbook and account manager
    - Atomic order placement with balance locking
    - Automatic trade execution and settlement
    - Price improvement detection and refunds
    - Market order cost estimation
    - Comprehensive error handling

#### HTTP API Layer
- **Endpoints**
    - `POST /api/v1/orders` - Place limit or market order
    - `POST /api/v1/orders/cancel` - Cancel existing order
    - `POST /api/v1/accounts/credit` - Add balance to account
    - `POST /api/v1/accounts/debit` - Remove balance from account
    - `GET /api/v1/accounts/balance` - Query account balances
    - `GET /api/v1/orderbook` - View orderbook state
    - `GET /health` - Health check endpoint

- **API Features**
    - RESTful design with proper HTTP methods
    - Comprehensive request validation
    - Structured error responses
    - JSON request/response format
    - Query parameter support
    - Swagger/OpenAPI documentation

#### Documentation
- **Swagger Integration**
    - Interactive API documentation at `/swagger/index.html`
    - Complete endpoint descriptions
    - Request/response examples
    - Schema definitions
    - Try-it-out functionality

- **Project Documentation**
    - Comprehensive README.md with:
        - Project overview and features
        - Architecture diagrams
        - Quick start guide
        - API endpoint documentation
        - Technical decision explanations
        - Testing guidelines
        - Docker deployment instructions
    - Postman collection for API testing

#### Infrastructure
- **Development Tools**
    - Makefile with common tasks (build, run, test, lint)
    - Docker support with multi-stage build
    - Docker Compose for easy deployment
    - Health checks in containers
    - Environment variable configuration

- **Testing**
    - Comprehensive unit tests for all components
    - Integration tests for complete workflows
    - Edge case coverage:
        - Insufficient balance scenarios
        - Self-trade prevention
        - Partial fills
        - Price improvement
        - Double cancellation
        - Market order liquidity checks
    - Test helpers for clean assertions

- **Logging**
    - Structured logging with levels (DEBUG, INFO, WARNING, ERROR)
    - Request timing metrics
    - Operation success/failure tracking
    - Error context preservation

#### Data Structures & Types
- **Order Types**
    - Limit orders with specific price
    - Market orders with immediate execution
    - Order states: Open, PartiallyFilled, Filled, Cancelled

- **Side Types**
    - Bid (Buy)
    - Ask (Sell)

- **Precision Handling**
    - Hybrid int64/float64 tick system
    - PriceToTicks/TicksToPrice conversion
    - Tick validation and normalization
    - Price tick: 0.01 BRL (1 cent)
    - Amount tick: 0.00000001 BTC (1 satoshi)

### Changed

#### Architecture Improvements
- Refactored from monolithic structure to clean architecture
- Separated concerns into distinct packages:
    - `internal/engine` - Matching engine orchestration
    - `internal/orderbook` - Order book logic
    - `internal/account` - Account management
    - `internal/handler` - HTTP handlers
    - `api/v1` - DTOs and request/response types
    - `pkg/logger` - Reusable logging
    - `pkg/utils` - Utility functions

#### Code Organization
- Moved from single files to organized package structure
- Created dedicated error types per package
- Extracted testing helpers to shared files
- Separated types, errors, and business logic

#### Performance Optimizations
- Used int64 for price comparisons to avoid float errors
- Implemented efficient limit level management
- Optimized sorting using slice indices

### Security

#### Input Validation
- Comprehensive request validation
- Price and amount range checks
- User ID and asset validation
- Pair format validation

#### Error Handling
- Explicit error returns (no panics)
- Detailed error messages
- Error context preservation
- Graceful degradation

### Technical Details

#### Dependencies
- Go 1.24
- Standard library HTTP server
- swaggo/swag v1.16.6 - Swagger generation
- swaggo/http-swagger v1.3.4 - Swagger UI

#### Configuration
- Environment variable support
- Configurable server address
- Default values for development

#### Deployment
- Docker image with optimized multi-stage build
- Health check integration
- Graceful shutdown support
- Production-ready containerization


#### Scalability Considerations
- Separate orderbooks per trading pair
- Lock granularity at orderbook level
- Read-optimized with RWMutex
- Memory-efficient data structures

### Known Limitations

#### Current Constraints
- In-memory storage (no persistence)
- Float64 precision limitations 
- Single-instance deployment
- No authentication/authorization
- No rate limiting

#### Future Improvements
- Database persistence (PostgreSQL)
- Redis caching for orderbook snapshots
- Migration to decimal.Decimal for precision
- WebSocket support for real-time updates
- JWT authentication
- Advanced order types (Stop-Loss, Stop-Limit, FOK, IOC)
- Metrics and monitoring
- Graceful shutdown implementation

---