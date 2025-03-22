package errors

type ValidationError struct {
	message string
}

type AuthError struct {
	message string
}

type NotFoundError struct {
	message string
}

type PaymentError struct {
	message string
	code    string
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	message string
}

// InternalServerError represents a server error
type InternalServerError struct {
	message string
}

// ForbiddenError represents a forbidden access error
type ForbiddenError struct {
	message string
}

// Implement error interface for all types
func (e *ValidationError) Error() string     { return e.message }
func (e *AuthError) Error() string           { return e.message }
func (e *NotFoundError) Error() string       { return e.message }
func (e *PaymentError) Error() string        { return e.message }
func (e *RateLimitError) Error() string      { return e.message }
func (e *InternalServerError) Error() string { return e.message }
func (e *ForbiddenError) Error() string      { return e.message }

// Add ErrorType methods for metrics tracking
func (e *ValidationError) ErrorType() string     { return "validation_error" }
func (e *AuthError) ErrorType() string           { return "auth_error" }
func (e *NotFoundError) ErrorType() string       { return "not_found_error" }
func (e *PaymentError) ErrorType() string        { return "payment_error" }
func (e *RateLimitError) ErrorType() string      { return "rate_limit_error" }
func (e *InternalServerError) ErrorType() string { return "internal_server_error" }
func (e *ForbiddenError) ErrorType() string      { return "forbidden_error" }

// Constructor functions
func NewValidationError(msg string) *ValidationError {
	return &ValidationError{message: msg}
}

func NewAuthError(msg string) *AuthError {
	return &AuthError{message: msg}
}

func NewNotFoundError(msg string) *NotFoundError {
	return &NotFoundError{message: msg}
}

func NewPaymentError(msg, code string) *PaymentError {
	return &PaymentError{message: msg, code: code}
}

func NewRateLimitError(message string) *RateLimitError {
	return &RateLimitError{message: message}
}

func NewInternalServerError(message string) *InternalServerError {
	return &InternalServerError{message: message}
}

func NewForbiddenError(message string) *ForbiddenError {
	return &ForbiddenError{message: message}
}
