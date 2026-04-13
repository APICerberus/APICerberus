package jsonutil

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const defaultReadLimitBytes int64 = 1 << 20 // 1MB

// WriteJSON writes a JSON response payload and status code.
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// ReadJSON decodes a JSON body into target with a body size limit.
func ReadJSON(r *http.Request, target any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = defaultReadLimitBytes
	}

	dec := json.NewDecoder(io.LimitReader(r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// MarshalJSON returns a compact JSON string, useful for SQLite text columns.
func MarshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("[WARN] json: MarshalJSON failed: %v, returning '{}'", err)
		return "{}"
	}
	return string(b)
}
