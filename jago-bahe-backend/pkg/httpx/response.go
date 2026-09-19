// Package httpx holds HTTP glue shared by every context's handlers: JSON
// response helpers, the error envelope, and middleware. Handlers convert DTOs
// here; they never JSON-serialize domain aggregates directly.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// maxBodyBytes caps a request body at 1 MiB: tight enough to blunt
// memory-exhaustion, and the reason the frontend compresses photos.
//
// A photo rides inline as a base64 data URL, which inflates it by ~4/3, so a
// phone-camera photo (2–5MB) does NOT fit here and never did — reporting one was
// a guaranteed 413 until the frontend downscaled first (see the frontend's
// lib/image.js, which budgets 900KB per photo against this cap). Raising this is
// the wrong lever: the cap is the guard, and the photo is what should be smaller.
const maxBodyBytes = 1 << 20

// ErrorBody is the single, consistent error envelope returned to clients. It is
// flat ({code, message}) so the frontend's response interceptor can read
// `data.message` directly. Domain sentinel errors are mapped to a code + message
// in one place (B1/B8); Postgres errors are never leaked.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes v as a JSON response with the given status code.
//
// A nil v is written as the JSON literal `null`, not as an empty body. The two
// endpoints that pass nil — GET /problems/{id}/obstacle and
// GET /problems/{id}/progress — both promise "200 with a null body, never a
// 404" (Scaffold §5), and an empty body does not keep that promise: it is not
// valid JSON at all, so a strict client fails to parse it and a lenient one
// (axios) silently yields "" instead of null. Callers that genuinely want no
// body should send 204 rather than come through here.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the standard error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorBody{Code: code, Message: message})
}

// Decode reads a JSON request body into dst, capping it at maxBodyBytes. It
// returns false (and writes the error) on malformed input (400) or an oversized
// body (413). Unknown fields are ignored (not rejected), keeping the lenient
// JSON contract §5 documents.
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(dst)
	if err == nil {
		// A request must contain exactly one JSON value. Reading to EOF also
		// enforces the size limit for trailing whitespace and extra content.
		var extra any
		if tailErr := dec.Decode(&extra); tailErr != io.EOF {
			if tailErr == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = tailErr
			}
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Error(w, http.StatusRequestEntityTooLarge, "body_too_large", "Request body is too large.")
			return false
		}
		Error(w, http.StatusBadRequest, "invalid_body", "Request body is not valid JSON.")
		return false
	}
	return true
}

// RequireFields writes a 400 and returns false if any named field is empty
// (after trimming). Fields is label→value; the label appears in the message so
// the client knows which field is missing. This is the single, consistent
// required-field check the write handlers share.
func RequireFields(w http.ResponseWriter, fields map[string]string) bool {
	for label, value := range fields {
		if strings.TrimSpace(value) == "" {
			Error(w, http.StatusBadRequest, "validation_error", label+" is required.")
			return false
		}
	}
	return true
}
