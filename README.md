# gokit

[![Go Test](https://github.com/piheta/gokit/actions/workflows/test.yml/badge.svg)](https://github.com/piheta/gokit/actions/workflows/test.yml)
[![Go Lint](https://github.com/piheta/gokit/actions/workflows/lint.yml/badge.svg)](https://github.com/piheta/gokit/actions/workflows/lint.yml)

A lightweight Go toolkit for building HTTP API services. Provides utilities for error handling, middleware support, and standardized API responses.

## Features

- **Error handling** - Structured API errors with HTTP status codes
- **Middleware support** - Chainable middleware for request processing
- **Response formatting** - Standardized API response structure

## Example Usage
```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/piheta/gokit"
)

func main() {
	mux := http.NewServeMux()

    // Register handlers using public middleware
	mux.Handle("GET /api/ping", gokit.Public(Ping))
	mux.Handle("GET /api/err", gokit.Public(Err))

	server := &http.Server{
		Addr:         ":8082",
		Handler:      gokit.RouterRequestLogger(mux), // Wrap handlers with logging middleware
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// Working handler
func Ping(w http.ResponseWriter, r *http.Request) error {
	return gokit.JSON(w, 200, "pong")
}

// Failing handler
func Err(w http.ResponseWriter, r *http.Request) error {
	err := gokit.NewError(404, "not_found", "user not found") // APIError (http code, type, message)

    // MetaErr, wraps error with additional key-value pair metadata for logging
	return gokit.WithMetadata(err, "user_id", "123", "email", "user@example.com") 
}
```

### Logging
```
2025/11/28 22:25:18 INFO REQ status=200 ms=0.09 ip=[::1]:51420 method=GET path=/api/ping
2025/11/28 22:25:24 WARN REQ status=404 ms=0.16 ip=[::1]:51425 method=GET path=/api/err error_detail="user not found" user_id=123 email=user@example.com error="Not Found"
```
