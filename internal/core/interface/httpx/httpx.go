// Package httpx holds transport-level HTTP helpers shared by every context's
// REST layer: a JSON writer, a problem-response writer, and a request-body
// decoder. It is pure mechanism — it knows nothing about any bounded context's
// errors, so each module maps its own domain errors to statuses and calls
// WriteProblem (rdt kept core ignorant of individual contexts the same way).
package httpx

import (
	"encoding/json"
	"net/http"
)

// Problem is the error response body: a stable machine-readable code plus a
// human-readable message. (A lean shape rather than full RFC 9457; revisit if a
// richer contract is needed.)
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteProblem writes a problem response with the given status.
func WriteProblem(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{Code: code, Message: message})
}

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON decodes the request body into dst, rejecting unknown fields. On
// failure it writes a 400 problem and returns false, so a handler can `if
// !httpx.DecodeJSON(...) { return }`.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return false
	}
	return true
}
