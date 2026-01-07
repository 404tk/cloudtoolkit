package schema

import (
	"errors"
	"fmt"
)

// Standard errors for provider operations
var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("not supported for this provider")
	ErrInvalidRegion  = errors.New("invalid region")
	ErrNoPermission   = errors.New("insufficient permissions")
	ErrInvalidConfig  = errors.New("invalid configuration")
)

// ProviderError wraps errors with provider context
type ProviderError struct {
	Provider string
	Action   string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Provider, e.Action, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new provider error
func NewProviderError(provider, action string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Action:   action,
		Err:      err,
	}
}
