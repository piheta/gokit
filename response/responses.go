// Package response provides HTTP response helpers for JSON and status code responses.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes the given data as JSON to the response writer with the specified status code.
func JSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

	return nil
}

// Status writes the HTTP status code without a response body.
func Status(w http.ResponseWriter, statusCode int) error {
	w.WriteHeader(statusCode)
	return nil
}
