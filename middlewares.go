// Package gokit provides utilities for building HTTP API services.
package gokit

import (
	"encoding/json"
	"net/http"
)

// APIFunc is a handler function that returns an error.
type APIFunc func(w http.ResponseWriter, r *http.Request) error

// Public wraps an APIFunc and converts returned errors to JSON responses with appropriate status codes.
func Public(h APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			err := MapError(err)

			w.WriteHeader(err.StatusCode)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				http.Error(w, "Error encoding response", http.StatusInternalServerError)
			}
		}
	}
}
