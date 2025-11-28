package apicore

import (
	"errors"
	"fmt"
)

type errMetadata struct {
	err      error // The wrapped error
	metadata []any // Key-value pairs compatible with slog
}

func (e *errMetadata) Error() string {
	return e.err.Error()
}

func (e *errMetadata) Unwrap() error {
	return e.err
}

// WithMetadata wraps an error with metadata key-value pairs for logging.
func WithMetadata(err error, pairs ...any) error {
	if err == nil {
		return nil
	}

	if len(pairs)%2 != 0 {
		pairs = pairs[:len(pairs)-1]
	}

	return &errMetadata{
		err:      err,
		metadata: pairs,
	}
}

// GetMetadata extracts all metadata key-value pairs from an error and its wrapped errors.
func GetMetadata(err error) []any {
	if err == nil {
		return nil
	}

	var allMetadata []any

	for err != nil {
		if metaErr, ok := err.(*errMetadata); ok {
			allMetadata = append(metaErr.metadata, allMetadata...)
		}
		err = errors.Unwrap(err)
	}

	return allMetadata
}

// GetMetadataMap extracts metadata as a map of string keys to any values.
func GetMetadataMap(err error) map[string]any {
	pairs := GetMetadata(err)
	if len(pairs) == 0 {
		return nil
	}

	result := make(map[string]any)

	for i := len(pairs) - 2; i >= 0; i -= 2 {
		if key, ok := pairs[i].(string); ok {
			result[key] = pairs[i+1]
		}
	}

	return result
}

// Wrap wraps an error with metadata key-value pairs.
func Wrap(err error, pairs ...any) error {
	return WithMetadata(err, pairs...)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf(format+": %w", append(args, err)...)
}

// HasMetadata checks if an error or any of its wrapped errors has metadata.
func HasMetadata(err error) bool {
	for err != nil {
		if _, ok := err.(*errMetadata); ok {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
