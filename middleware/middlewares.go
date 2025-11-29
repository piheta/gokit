// Package middleware provides HTTP middleware utilities for API request handling and logging.
package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/piheta/apicore/apierr"
	"github.com/piheta/apicore/metaerr"
)

// APIFunc is a handler function that returns an error.
type APIFunc func(w http.ResponseWriter, r *http.Request) error

// Public wraps an APIFunc and converts returned errors to JSON responses with appropriate status codes.
func Public(h APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			err := apierr.MapError(err, r)

			w.WriteHeader(err.StatusCode)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				http.Error(w, "Error encoding response", http.StatusInternalServerError)
			}
		}
	}
}

// RouterRequestLogger logs HTTP requests with method, path, status, and duration.
func RouterRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rr, r)

		method := r.Method
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		ip := r.RemoteAddr
		status := rr.statusCode

		if !strings.HasPrefix(path, "/api") {
			return
		}
		if method == "OPTIONS" {
			return
		}

		duration := time.Since(start)
		durationMs := float64(duration.Microseconds()) / 1000

		attrs := []any{
			slog.Int("status", status),
			slog.String("ms", fmt.Sprintf("%.2f", durationMs)),
			slog.String("ip", ip),
			slog.String("method", method),
			slog.String("path", path),
		}

		// Log based on status code
		if status >= http.StatusBadRequest {
			// Include original error details and metadata if available
			if originalErr, ok := r.Context().Value(apierr.OriginalErrorContextKey).(error); ok {
				attrs = append(attrs, slog.String("error_detail", originalErr.Error()))

				// Add structured metadata from the original error
				metadata := metaerr.GetMetadata(originalErr)
				attrs = append(attrs, metadata...)
			}

			errMsg := http.StatusText(status)
			attrs = append(attrs, slog.String("error", errMsg))

			if status >= http.StatusInternalServerError {
				slog.Error("REQ", attrs...)
			} else {
				slog.Warn("REQ", attrs...)
			}
		} else {
			slog.Info("REQ", attrs...)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) Flush() {
	if flusher, ok := rr.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}
