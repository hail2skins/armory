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

// Implement error interface for all types
func (e *ValidationError) Error() string { return e.message }
func (e *AuthError) Error() string       { return e.message }
func (e *NotFoundError) Error() string   { return e.message }
func (e *PaymentError) Error() string    { return e.message }

// Add ErrorType methods for metrics tracking
func (e *ValidationError) ErrorType() string { return "validation_error" }
func (e *AuthError) ErrorType() string       { return "auth_error" }
func (e *NotFoundError) ErrorType() string   { return "not_found_error" }
func (e *PaymentError) ErrorType() string    { return "payment_error" }

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
