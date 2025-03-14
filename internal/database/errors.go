package database

import "errors"

var (
	// ErrInvalidToken is returned when a token is invalid or expired
	ErrInvalidToken = errors.New("invalid or expired token")
)
