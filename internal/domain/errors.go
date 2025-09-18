package domain

import (
	"net/http"
)

type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *DomainError) Error() string {
	return e.Message
}

var (
	ErrOrderNotFound = &DomainError{
		Code:    "ORDER_NOT_FOUND",
		Message: "Order not found",
		Status:  http.StatusNotFound,
	}

	ErrOrderNotOpen = &DomainError{
		Code:    "ORDER_NOT_OPEN",
		Message: "Order is not in open status",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidOrder = &DomainError{
		Code:    "INVALID_ORDER",
		Message: "Invalid order data",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidSymbol = &DomainError{
		Code:    "INVALID_SYMBOL",
		Message: "Invalid trading symbol",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidQuantity = &DomainError{
		Code:    "INVALID_QUANTITY",
		Message: "Quantity must be greater than 0",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidPrice = &DomainError{
		Code:    "INVALID_PRICE",
		Message: "Price must be greater than 0 for limit orders",
		Status:  http.StatusBadRequest,
	}

	ErrDatabaseError = &DomainError{
		Code:    "DATABASE_ERROR",
		Message: "Database operation failed",
		Status:  http.StatusInternalServerError,
	}

	ErrOrderProcessingFailed = &DomainError{
		Code:    "ORDER_PROCESSING_FAILED",
		Message: "Failed to process order",
		Status:  http.StatusInternalServerError,
	}
)

// NewDomainError creates a new domain error
func NewDomainError(code, message string, status int) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// IsDomainError checks if an error is a domain error
func IsDomainError(err error) (*DomainError, bool) {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr, true
	}
	return nil, false
}
