package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"jago-bahe-backend/pkg/httpx"
)

// TestJSONNilWritesNullNotAnEmptyBody pins the "null-shaped, never not-found"
// contract at its source.
//
// GET /problems/{id}/obstacle and GET /problems/{id}/progress both answer 200
// with a null body when there is nothing to report — that uniformity is what
// stops the difference between "no case" and "hidden problem" from being
// observable. JSON() used to skip the body entirely for a nil v, which sends
// Content-Length: 0. That is not valid JSON: a strict client errors, and axios
// quietly yields "" instead of null, so the frontend's null check would pass by
// accident rather than by contract.
func TestJSONNilWritesNullNotAnEmptyBody(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.JSON(rec, http.StatusOK, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "null\n" {
		t.Errorf("body = %q, want %q — an empty body is not valid JSON", got, "null\n")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestJSONEncodesAValue(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.JSON(rec, http.StatusCreated, map[string]string{"id": "prob-1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if got, want := rec.Body.String(), "{\"id\":\"prob-1\"}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}
