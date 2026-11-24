// Package httpx holds HTTP helpers shared by the server layer: request
// context keys, JSON responses and domain-error mapping. It is a leaf
// package so every handler subpackage can use it without cycles.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	domainerrors "github.com/radosny/vcoach/server/internal/errors"
)

type athleteIDKey struct{}

// WithAthleteID stores the resolved athlete in the request context.
func WithAthleteID(ctx context.Context, athleteID uuid.UUID) context.Context {
	return context.WithValue(ctx, athleteIDKey{}, athleteID)
}

// AthleteID returns the athlete resolved for the request.
func AthleteID(ctx context.Context) uuid.UUID {
	athleteID, _ := ctx.Value(athleteIDKey{}).(uuid.UUID)
	return athleteID
}

// WriteJSON writes a JSON response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status code is already sent; nothing left to do but report.
		http.Error(w, "encoding response failed", http.StatusInternalServerError)
	}
}

// WriteError maps an error to an HTTP error response.
func WriteError(w http.ResponseWriter, err error) {
	status, message := MapServiceError(err)
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// ErrorResponse is the error envelope of every failing endpoint.
type ErrorResponse struct {
	Error string `json:"error" example:"validation failed"`
}

// MapServiceError converts domain errors into an HTTP status and a
// user-presentable message. Unknown errors become 500 without detail.
func MapServiceError(err error) (int, string) {
	var validationErr *domainerrors.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest, validationErr.Error()
	}
	var notFoundErr *domainerrors.NotFoundError
	if errors.As(err, &notFoundErr) {
		return http.StatusNotFound, notFoundErr.Error()
	}
	var conflictErr *domainerrors.ConflictError
	if errors.As(err, &conflictErr) {
		return http.StatusConflict, conflictErr.Error()
	}
	return http.StatusInternalServerError, "internal error"
}
