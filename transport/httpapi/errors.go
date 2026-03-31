package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/flames-hq/flames/provider/providererr"
)

type errorResponse struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
}

func errorToStatus(err error) int {
	switch {
	case errors.Is(err, providererr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, providererr.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, providererr.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := errorToStatus(err)

	resp := errorResponse{
		Code:    http.StatusText(status),
		Message: err.Error(),
	}

	var pe *providererr.ProviderError
	if errors.As(err, &pe) {
		resp.Code = pe.Code
		resp.Message = pe.Message
		resp.ResourceType = pe.ResourceType
		resp.ResourceID = pe.ResourceID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// writeErrorMessage writes a structured JSON error response with a code and message.
func writeErrorMessage(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		Code:    code,
		Message: message,
	})
}
