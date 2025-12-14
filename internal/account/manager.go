package account

import "sync"

type Manager struct {
	accounts map[string]map[string]*Balance
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		accounts: make(map[string]map[string]*Balance),
	}
}

// Credit adds amount to available balance
func (m *Manager) Credit(userID, asset string, amount float64) error {
	// Validate User, Asset and Amount
	err := m.validateInputs(userID, asset, amount)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	balance := m.getOrCreateBalance(userID, asset)
	balance.Available += amount

	return nil
}

// Debit remove amount from available balance
func (m *Manager) Debit(userID, asset string, amount float64) error {
	// Validate User, Asset and Amount
	err := m.validateInputs(userID, asset, amount)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	balance := m.getOrCreateBalance(userID, asset)
	if balance.Available < amount {
		return ErrInsufficientBalance
	}

	balance.Available -= amount
	return nil
}

// Lock amount from available balance to locked
func (m *Manager) Lock(userID, asset string, amount float64) error {
	// Validate User, Asset and Amount
	err := m.validateInputs(userID, asset, amount)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	balance := m.getOrCreateBalance(userID, asset)
	if balance.Available < amount {
		return ErrInsufficientBalance
	}

	balance.Available -= amount
	balance.Locked += amount
	return nil
}

// Unlock amount from locked to available
func (m *Manager) Unlock(userID, asset string, amount float64) error {
	// Validate User, Asset and Amount
	err := m.validateInputs(userID, asset, amount)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	balance := m.getOrCreateBalance(userID, asset)
	if balance.Locked < amount {
		return ErrInsufficientLocked
	}

	balance.Locked -= amount
	balance.Available += amount
	return nil
}

// DebitLocked remove amount from locked balance
func (m *Manager) DebitLocked(userID, asset string, amount float64) error {
	// Validate User, Asset and Amount
	err := m.validateInputs(userID, asset, amount)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	balance := m.getOrCreateBalance(userID, asset)
	if balance.Locked < amount {
		return ErrInsufficientLocked
	}

	balance.Locked -= amount
	return nil
}

func (m *Manager) GetBalance(userID, asset string) *Balance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if userAssets, exists := m.accounts[userID]; exists {
		if balance, exists := userAssets[asset]; exists {
			// Return copy to avoid external modification
			return &Balance{
				Available: balance.Available,
				Locked:    balance.Locked,
			}
		}
	}
	return nil
}

func (m *Manager) GetAllBalances(userID string) map[string]*Balance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Balance)

	if userAssets, exists := m.accounts[userID]; exists {
		for asset, balance := range userAssets {
			result[asset] = &Balance{
				Available: balance.Available,
				Locked:    balance.Locked,
			}
		}
	}

	return result
}

func (m *Manager) getOrCreateBalance(userID, asset string) *Balance {
	if _, exists := m.accounts[userID]; !exists {
		m.accounts[userID] = make(map[string]*Balance)
	}

	if _, exists := m.accounts[userID][asset]; !exists {
		m.accounts[userID][asset] = &Balance{}
	}

	return m.accounts[userID][asset]
}

func (m *Manager) validateInputs(userID, asset string, amount float64) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if asset == "" {
		return ErrInvalidAsset
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}
