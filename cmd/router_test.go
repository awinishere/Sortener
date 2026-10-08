package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupRouter(t *testing.T) http.Handler {
	t.Helper()
	store, _ := setupStore(t)
	handler := &Handler{store: store, baseUrl: "http://short.test"}
	return NewRouter(handler)
}

func TestRouterShortenThenRedirect(t *testing.T) {
	router := setupRouter(t)

	postReq := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	postRec := httptest.NewRecorder()
	router.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusCreated {
		t.Fatalf("POST /shorten status = %d, want %d, body = %q", postRec.Code, http.StatusCreated, postRec.Body)
	}

	var resp shortenResponse
	if err := json.NewDecoder(postRec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusFound {
		t.Fatalf("GET /%s status = %d, want %d", resp.Code, getRec.Code, http.StatusFound)
	}
	if loc := getRec.Header().Get("Location"); loc != "https://example.com" {
		t.Errorf("Location = %q, want %q", loc, "https://example.com")
	}
}

func TestRouterUnknownCode(t *testing.T) {
	router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/xyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /xyz status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	router := setupRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"put to shorten", http.MethodPut, "/shorten"},
		{"delete to shorten", http.MethodDelete, "/shorten"},
		{"post to code path", http.MethodPost, "/abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestRouterGetShortenMatchesWildcard(t *testing.T) {
	router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /shorten status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouterShortenInvalidBody(t *testing.T) {
	router := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`no json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /shorten with invalid body status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
