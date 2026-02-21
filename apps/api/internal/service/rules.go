package service

import (
	"errors"
	"strings"
)

var (
	ErrInvalidCartOwner = errors.New("exactly one of user_id or session_id must be provided")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrMissingShipping  = errors.New("shipping fields are required when order is shipped")
)

type ShippingAddress struct {
	Name        string
	AddressLine string
	City        string
	PostalCode  string
	Country     string
}

func ValidateCartOwner(userIDPresent bool, sessionID string) error {
	hasSession := strings.TrimSpace(sessionID) != ""
	if userIDPresent == hasSession {
		return ErrInvalidCartOwner
	}
	return nil
}

func ValidateOrderStatus(status string) error {
	switch status {
	case "pending", "paid", "failed", "refunded", "shipped":
		return nil
	default:
		return ErrInvalidStatus
	}
}

func ValidateShippingForStatus(status string, addr ShippingAddress) error {
	if status != "shipped" {
		return nil
	}
	if strings.TrimSpace(addr.Name) == "" ||
		strings.TrimSpace(addr.AddressLine) == "" ||
		strings.TrimSpace(addr.City) == "" ||
		strings.TrimSpace(addr.PostalCode) == "" ||
		strings.TrimSpace(addr.Country) == "" {
		return ErrMissingShipping
	}
	return nil
}
