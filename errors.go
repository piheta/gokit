// Package gokit provides utilities for building HTTP API services.
package gokit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

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
func MapError(err error) *APIError {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*APIError); ok {
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

	if errors.Is(err, context.Canceled) {
		return NewError(499, "canceled", "request cancelled")
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return NewError(504, "canceled", "request timeout")
	}

	slog.With("error", err).Error("Error missed mappers!")
	return NewError(500, "internal", "internal server error")
}
