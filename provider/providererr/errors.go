// Package providererr defines structured error types shared across all provider implementations.
package providererr

import "fmt"

// Sentinel errors for use with errors.Is matching.
var (
	ErrNotFound      = &ProviderError{Code: "not_found"}
	ErrAlreadyExists = &ProviderError{Code: "already_exists"}
	ErrConflict      = &ProviderError{Code: "conflict"}
	ErrNotSupported  = &ProviderError{Code: "not_supported"}
	ErrCacheMiss     = &ProviderError{Code: "cache_miss"}
	ErrNoJobs        = &ProviderError{Code: "no_jobs"}
)

// ProviderError carries structured metadata about a provider failure, including a
// machine-readable code, the affected resource type and ID, and an optional wrapped cause.
type ProviderError struct {
	Code         string
	Message      string
	ResourceType string
	ResourceID   string
	Err          error
}

// Error returns a human-readable description of the error.
func (e *ProviderError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.ResourceType != "" && e.ResourceID != "" {
		return fmt.Sprintf("%s: %s %s", e.Code, e.ResourceType, e.ResourceID)
	}
	return e.Code
}

// Unwrap returns the underlying cause, supporting errors.Unwrap.
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// Is reports whether the error matches target by comparing error codes, supporting errors.Is.
func (e *ProviderError) Is(target error) bool {
	t, ok := target.(*ProviderError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NotFound returns an ErrNotFound error with the given resource type and ID.
func NotFound(resourceType, resourceID string) error {
	return &ProviderError{
		Code:         "not_found",
		Message:      fmt.Sprintf("%s not found: %s", resourceType, resourceID),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// AlreadyExists returns an ErrAlreadyExists error with the given resource type and ID.
func AlreadyExists(resourceType, resourceID string) error {
	return &ProviderError{
		Code:         "already_exists",
		Message:      fmt.Sprintf("%s already exists: %s", resourceType, resourceID),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// Conflict returns an ErrConflict error with the given resource type, ID, and description.
func Conflict(resourceType, resourceID, message string) error {
	return &ProviderError{
		Code:         "conflict",
		Message:      message,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}
