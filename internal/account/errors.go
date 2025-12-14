package account

import "errors"

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInsufficientLocked  = errors.New("insufficient locked balance")
	ErrInvalidAmount       = errors.New("amount must be greater than 0")
	ErrInvalidAsset        = errors.New("asset cannot be empty")
	ErrInvalidUserID       = errors.New("userID cannot be empty")
)
