package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piheta/apicore/response"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		data           any
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "success with struct",
			statusCode:     200,
			data:           map[string]string{"key": "value"},
			expectedStatus: 200,
			expectedType:   "application/json",
		},
		{
			name:           "created",
			statusCode:     201,
			data:           map[string]int{"id": 123},
			expectedStatus: 201,
			expectedType:   "application/json",
		},
		{
			name:           "bad request",
			statusCode:     400,
			data:           map[string]string{"error": "invalid input"},
			expectedStatus: 400,
			expectedType:   "application/json",
		},
		{
			name:           "array response",
			statusCode:     200,
			data:           []map[string]any{{"id": 1}, {"id": 2}},
			expectedStatus: 200,
			expectedType:   "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			err := response.JSON(w, tt.statusCode, tt.data)

			if err != nil {
				t.Errorf("JSON() returned error: %v", err)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %q, want %q", contentType, tt.expectedType)
			}

			var result any
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Errorf("Response body is not valid JSON: %v", err)
			}
		})
	}
}

func TestJSONEncoding(t *testing.T) {
	w := httptest.NewRecorder()

	testData := map[string]any{
		"name": "test",
		"age":  25,
		"tags": []string{"go", "testing"},
	}

	_ = response.JSON(w, http.StatusOK, testData)

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name=test, got %v", result["name"])
	}
	if result["age"] != float64(25) {
		t.Errorf("Expected age=25, got %v", result["age"])
	}
}

func BenchmarkJSON(b *testing.B) {
	data := map[string]any{
		"id":    123,
		"name":  "test",
		"email": "test@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_ = response.JSON(w, http.StatusOK, data)
	}
}

func TestJSONWriteError(_ *testing.T) {
	// Create a response writer that fails on Write
	failingWriter := &failingResponseWriter{}

	// This should not panic even if encoding fails
	_ = response.JSON(failingWriter, http.StatusOK, map[string]string{"key": "value"})
}

// failingResponseWriter is a mock ResponseWriter that fails on Write
type failingResponseWriter struct {
	headerWritten bool
}

func (f *failingResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (f *failingResponseWriter) Write(_ []byte) (int, error) {
	return 0, bytes.ErrTooLarge
}

func (f *failingResponseWriter) WriteHeader(_ int) {
	f.headerWritten = true
}
