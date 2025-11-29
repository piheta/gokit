package tests

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entrans "github.com/go-playground/validator/v10/translations/en"

	"github.com/piheta/apicore/apierr"
	"github.com/piheta/apicore/metaerr"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   *apierr.APIError
		expected string
	}{
		{
			name: "string message",
			apiErr: &apierr.APIError{
				StatusCode: 400,
				Type:       "parameter",
				Message:    "invalid parameter",
			},
			expected: "invalid parameter",
		},
		{
			name: "map message",
			apiErr: &apierr.APIError{
				StatusCode: 400,
				Type:       "validation",
				Message: map[string]any{
					"field": "email",
					"error": "invalid format",
				},
			},
			expected: `{"error":"invalid format","field":"email"}`,
		},
		{
			name: "number message",
			apiErr: &apierr.APIError{
				StatusCode: 500,
				Type:       "internal",
				Message:    42,
			},
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiErr.Error()
			if tt.name == "map message" {
				// For map messages, just check it's valid JSON
				var m map[string]any
				if err := json.Unmarshal([]byte(result), &m); err != nil {
					t.Errorf("Error() returned invalid JSON: %v", err)
				}
			} else if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAPIError_Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "400", statusCode: 400},
		{name: "404", statusCode: 404},
		{name: "500", statusCode: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &apierr.APIError{StatusCode: tt.statusCode}
			if got := apiErr.Status(); got != tt.statusCode {
				t.Errorf("Status() = %d, want %d", got, tt.statusCode)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		errtype        string
		message        any
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "string message",
			code:           400,
			errtype:        "parameter",
			message:        "invalid param",
			expectedStatus: 400,
			expectedType:   "parameter",
		},
		{
			name:           "map message",
			code:           422,
			errtype:        "validation",
			message:        map[string]any{"field": "email"},
			expectedStatus: 422,
			expectedType:   "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apierr.NewError(tt.code, tt.errtype, tt.message)
			if err.Status() != tt.expectedStatus {
				t.Errorf("Status() = %d, want %d", err.Status(), tt.expectedStatus)
			}
			if err.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", err.Type, tt.expectedType)
			}
		})
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedStatus: 0,
			expectedType:   "",
		},
		{
			name: "existing APIError",
			err: &apierr.APIError{
				StatusCode: 403,
				Type:       "forbidden",
				Message:    "access denied",
			},
			expectedStatus: 403,
			expectedType:   "forbidden",
		},
		{
			name:           "JSON syntax error",
			err:            &json.SyntaxError{Offset: 5},
			expectedStatus: 400,
			expectedType:   "json",
		},
		{
			name:           "EOF error",
			err:            io.EOF,
			expectedStatus: 400,
			expectedType:   "json",
		},
		{
			name:           "unexpected EOF",
			err:            io.ErrUnexpectedEOF,
			expectedStatus: 400,
			expectedType:   "json",
		},
		{
			name:           "context cancelled",
			err:            context.Canceled,
			expectedStatus: 499,
			expectedType:   "canceled",
		},
		{
			name:           "context deadline exceeded",
			err:            context.DeadlineExceeded,
			expectedStatus: 504,
			expectedType:   "canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := apierr.MapError(tt.err, nil)
			if tt.err == nil {
				if result != nil {
					t.Errorf("MapError(nil) should return nil, got %v", result)
				}
				return
			}
			if result.Status() != tt.expectedStatus {
				t.Errorf("Status() = %d, want %d", result.Status(), tt.expectedStatus)
			}
			if result.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", result.Type, tt.expectedType)
			}
		})
	}
}

func TestMapError_UnmarshalTypeError(t *testing.T) {
	// Test actual JSON unmarshal type error by decoding invalid JSON
	invalidJSON := `{"age": "not a number"}`
	var data struct {
		Age int `json:"age"`
	}

	err := json.Unmarshal([]byte(invalidJSON), &data)
	if err == nil {
		t.Fatal("Expected unmarshal error, got nil")
	}

	result := apierr.MapError(err, nil)
	if result.Status() != 400 {
		t.Errorf("Status() = %d, want 400", result.Status())
	}
	if result.Type != "json" {
		t.Errorf("Type = %q, want json", result.Type)
	}
}

func TestMapError_WithMetadata(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedType   string
		expectedMsg    string
	}{
		{
			name:           "APIError wrapped with metadata",
			err:            metaerr.WithMetadata(apierr.NewError(401, "not_found", "user not found"), "user_id", "123"),
			expectedStatus: 401,
			expectedType:   "not_found",
			expectedMsg:    "user not found",
		},
		{
			name:           "APIError wrapped with multiple metadata",
			err:            metaerr.WithMetadata(metaerr.WithMetadata(apierr.NewError(403, "forbidden", "access denied"), "user_id", "123"), "resource", "admin"),
			expectedStatus: 403,
			expectedType:   "forbidden",
			expectedMsg:    "access denied",
		},
		{
			name:           "unwrapped APIError still works",
			err:            apierr.NewError(404, "not_found", "resource not found"),
			expectedStatus: 404,
			expectedType:   "not_found",
			expectedMsg:    "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := apierr.MapError(tt.err, nil)
			if result.Status() != tt.expectedStatus {
				t.Errorf("Status() = %d, want %d", result.Status(), tt.expectedStatus)
			}
			if result.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", result.Type, tt.expectedType)
			}
			if result.Error() != tt.expectedMsg {
				t.Errorf("Error() = %q, want %q", result.Error(), tt.expectedMsg)
			}
		})
	}
}

func TestMapError_ValidationError(t *testing.T) {
	v := validator.New()

	type User struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}

	// Test with missing required fields
	user := User{}
	err := v.Struct(user)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	result := apierr.MapError(err, nil)
	if result.Status() != 422 {
		t.Errorf("Status() = %d, want 422", result.Status())
	}
	if result.Type != "validation" {
		t.Errorf("Type = %q, want validation", result.Type)
	}

	// Check that message is a map with field errors
	fieldErrors, ok := result.Message.(map[string]string)
	if !ok {
		t.Errorf("Message should be map[string]string, got %T", result.Message)
	}

	// Should have errors for both email and age fields
	if _, hasEmail := fieldErrors["email"]; !hasEmail {
		t.Error("Expected email field error, got none")
	}
	if _, hasAge := fieldErrors["age"]; !hasAge {
		t.Error("Expected age field error, got none")
	}
}

func TestMapError_ValidationError_WithTranslator(t *testing.T) {
	// Set up translator
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")

	v := validator.New()
	if err := entrans.RegisterDefaultTranslations(v, trans); err != nil {
		t.Fatalf("Failed to register translations: %v", err)
	}

	// Set the translator in apicore
	apierr.SetTranslator(trans)
	defer apierr.SetTranslator(nil) // Clean up after test

	type User struct {
		Email string `validate:"required,email"`
	}

	user := User{Email: "invalid"} // Invalid email format
	err := v.Struct(user)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	result := apierr.MapError(err, nil)
	if result.Status() != 422 {
		t.Errorf("Status() = %d, want 422", result.Status())
	}

	fieldErrors, ok := result.Message.(map[string]string)
	if !ok {
		t.Errorf("Message should be map[string]string, got %T", result.Message)
	}

	// With translator, should have translated message (not just tag name)
	if msg, hasEmail := fieldErrors["email"]; !hasEmail {
		t.Error("Expected email field error, got none")
	} else if msg == "email" {
		t.Error("Expected translated message, got just tag name")
	}
}

func TestMapError_ValidationError_NoTranslator(t *testing.T) {
	// Make sure translator is not set
	apierr.SetTranslator(nil)

	v := validator.New()

	type Product struct {
		Name string `validate:"required"`
	}

	product := Product{} // Missing required field
	err := v.Struct(product)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	result := apierr.MapError(err, nil)
	if result.Status() != 422 {
		t.Errorf("Status() = %d, want 422", result.Status())
	}

	fieldErrors, ok := result.Message.(map[string]string)
	if !ok {
		t.Errorf("Message should be map[string]string, got %T", result.Message)
	}

	// Without translator, should just have the tag name
	if msg, hasName := fieldErrors["name"]; !hasName {
		t.Error("Expected name field error, got none")
	} else if msg != "required" {
		t.Errorf("Expected 'required' tag, got %q", msg)
	}
}

func TestMapError_ValidationError_NestedFields(t *testing.T) {
	v := validator.New()

	type Address struct {
		City string `validate:"required"`
	}

	type Person struct {
		Name    string  `validate:"required"`
		Address Address `validate:"required"`
	}

	person := Person{Address: Address{}} // Missing nested field
	err := v.Struct(person)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	result := apierr.MapError(err, nil)
	fieldErrors, ok := result.Message.(map[string]string)
	if !ok {
		t.Errorf("Message should be map[string]string, got %T", result.Message)
	}

	// Should extract just the nested field name (not the full namespace)
	if _, hasCity := fieldErrors["city"]; !hasCity {
		t.Error("Expected city field error, got none")
	}
}
