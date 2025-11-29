// Package apierr provides API error types and error mapping utilities.
package apierr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var defaultTranslator ut.Translator

// SetTranslator sets the translator for validation error messages.
// Call this once during app initialization if you want translated error messages.
func SetTranslator(trans ut.Translator) {
	defaultTranslator = trans
}

type contextKey string

// OriginalErrorContextKey is the key for storing the original error in request context.
const OriginalErrorContextKey contextKey = "OriginalError"

// APIError represents an API error with HTTP status code, type, and message.
type APIError struct {
	StatusCode int    `json:"status"` // HTTP status code
	Type       string `json:"type"`
	Message    any    `json:"msg"` // Support various message types
}

func (e *APIError) Error() string {
	switch msg := e.Message.(type) {
	case string:
		return msg
	case map[string]any:
		jsonBytes, err := json.Marshal(msg)
		if err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("error marshaling message: %v", err)
	default:
		return fmt.Sprintf("%v", msg)
	}
}

// Status returns the HTTP status code of the error.
func (e *APIError) Status() int {
	return e.StatusCode
}

// NewError creates a new APIError with the given status code, type, and message.
func NewError(code int, errtype string, message any) *APIError {
	if messageFmt, ok := message.(string); ok {
		message = messageFmt
	}
	return &APIError{
		StatusCode: code,
		Type:       errtype,
		Message:    message,
	}
}

// MapError converts various error types to APIError with appropriate HTTP status codes and messages.
func MapError(err error, r *http.Request) *APIError {
	if err == nil {
		return nil
	}

	// Store the original error in context for RequestLogger
	// It will log the metadata
	if r != nil {
		ctx := context.WithValue(r.Context(), OriginalErrorContextKey, err)
		*r = *r.WithContext(ctx)
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}

	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &unmarshalErr) {
		return NewError(400, "json", "invalid JSON format")
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return NewError(400, "json", "empty or incomplete JSON body")
	}

	var validationErr validator.ValidationErrors
	if errors.As(err, &validationErr) {
		formattedErrors := formatValidationErrors(validationErr)
		return NewError(422, "validation", formattedErrors)
	}

	if errors.Is(err, context.Canceled) {
		return NewError(499, "canceled", "request cancelled")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return NewError(504, "canceled", "request timeout")
	}

	slog.With("error", err).Error("Error missed mappers!")
	return NewError(500, "internal", "internal server error")
}

func formatValidationErrors(validationErrors validator.ValidationErrors) map[string]string {
	formattedErrors := make(map[string]string)

	// If translator is set, use it. Otherwise, just use the tag names.
	if defaultTranslator != nil {
		translatedErrors := validationErrors.Translate(defaultTranslator)
		for fieldError, translatedError := range translatedErrors {
			parts := strings.Split(fieldError, ".")
			fieldName := strings.ToLower(parts[len(parts)-1])
			formattedErrors[fieldName] = translatedError
		}
	} else {
		// Fallback: just use validation tags without translation
		for _, err := range validationErrors {
			fieldName := strings.ToLower(err.Field())
			formattedErrors[fieldName] = err.Tag()
		}
	}

	return formattedErrors
}
