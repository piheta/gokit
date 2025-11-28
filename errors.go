package gokit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type ApiError struct {
	StatusCode int    `json:"status"` // HTTP status code
	Type       string `json:"type"`
	Message    any    `json:"msg"` // Support various message types
}

func (e *ApiError) Error() string {
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

func (e *ApiError) Status() int {
	return e.StatusCode
}

func NewError(code int, errtype string, message any) *ApiError {
	if messageFmt, ok := message.(string); ok {
		message = messageFmt
	}
	return &ApiError{
		StatusCode: code,
		Type:       errtype,
		Message:    message,
	}
}

func MapError(err error) *ApiError {
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*ApiError); ok {
		return apiErr
	}

	if strings.Contains(err.Error(), "invalid UUID") {
		return NewError(400, "parameter", "invalid parameter")
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
