package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piheta/gokit"
)

func TestPublic_SuccessfulHandler(t *testing.T) {
	handler := gokit.Public(func(w http.ResponseWriter, _ *http.Request) error {
		_ = gokit.JSON(w, http.StatusOK, map[string]string{"message": "success"})
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message=success, got %v", result["message"])
	}
}

func TestPublic_APIErrorHandler(t *testing.T) {
	handler := gokit.Public(func(_ http.ResponseWriter, _ *http.Request) error {
		return gokit.NewError(http.StatusBadRequest, "validation", "invalid input")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result gokit.APIError
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Type != "validation" {
		t.Errorf("Expected type=validation, got %q", result.Type)
	}
	if result.Message != "invalid input" {
		t.Errorf("Expected message=invalid input, got %v", result.Message)
	}
}

func TestPublic_UnmappedErrorHandler(t *testing.T) {
	handler := gokit.Public(func(_ http.ResponseWriter, _ *http.Request) error {
		return io.ErrUnexpectedEOF
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	handler(w, r)

	// UnmappedError should map to JSON error
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result gokit.APIError
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Type != "json" {
		t.Errorf("Expected type=json, got %q", result.Type)
	}
}

func TestPublic_RequestCancelledHandler(t *testing.T) {
	handler := gokit.Public(func(_ http.ResponseWriter, _ *http.Request) error {
		return gokit.NewError(http.StatusBadRequest, "test", "")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r)

	var result gokit.APIError
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPublic_HeaderSet(t *testing.T) {
	handler := gokit.Public(func(w http.ResponseWriter, _ *http.Request) error {
		_ = gokit.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %q", contentType)
	}
}

func TestPublic_MultipleHandlerCalls(t *testing.T) {
	callCount := 0
	handler := gokit.Public(func(w http.ResponseWriter, _ *http.Request) error {
		callCount++
		_ = gokit.JSON(w, http.StatusOK, map[string]int{"count": callCount})
		return nil
	})

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		handler(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Call %d: Expected status %d, got %d", i, http.StatusOK, w.Code)
		}
	}

	if callCount != 3 {
		t.Errorf("Expected handler to be called 3 times, was called %d times", callCount)
	}
}

func TestPublic_ComplexErrorMessage(t *testing.T) {
	handler := gokit.Public(func(_ http.ResponseWriter, _ *http.Request) error {
		return gokit.NewError(
			http.StatusUnprocessableEntity,
			"validation",
			map[string]any{
				"fields": []string{"email", "password"},
				"reason": "missing required fields",
			},
		)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	handler(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var result gokit.APIError
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Type != "validation" {
		t.Errorf("Expected type=validation, got %q", result.Type)
	}
}

func BenchmarkPublic(b *testing.B) {
	handler := gokit.Public(func(w http.ResponseWriter, _ *http.Request) error {
		_ = gokit.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		handler(w, r)
	}
}
