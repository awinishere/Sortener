package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func setupHandler(t *testing.T) (*Handler, *miniredis.Miniredis) {
	t.Helper()
	store, mr := setupStore(t)
	return &Handler{store: store, baseUrl: "http://short.test"}, mr
}

func TestValidUrl(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com/path?a=1", true},
		{"not a url", false},
		{"ftp://example.com", false},
		{"https://", false},
		{"/relative/path", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := validUrl(tt.in); got != tt.want {
			t.Errorf("validUrl(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestShortenSuccess(t *testing.T) {
	handler, _ := setupHandler(t)

	body := `{"url":"https://example.com/page"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Shorten(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %q", rec.Code, http.StatusCreated, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp shortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code == "" {
		t.Fatal("response code is empty")
	}
	if want := "http://short.test/" + resp.Code; resp.ShortUrl != want {
		t.Errorf("short_url = %q, want %q", resp.ShortUrl, want)
	}

	redirectReq := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	redirectReq.SetPathValue("code", resp.Code)
	redirectRec := httptest.NewRecorder()

	handler.Redirect(redirectRec, redirectReq)

	if redirectRec.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want %d", redirectRec.Code, http.StatusFound)
	}
	if loc := redirectRec.Header().Get("Location"); loc != "https://example.com/page" {
		t.Errorf("Location = %q, want %q", loc, "https://example.com/page")
	}
}

func TestShortenBadRequest(t *testing.T) {
	handler, _ := setupHandler(t)

	tests := []struct {
		name string
		body string
	}{
		{"not json", `{"url": `},
		{"empty body", ``},
		{"missing url field", `{"foo":"bar"}`},
		{"invalid url", `{"url":"ftp://example.com"}`},
		{"body too large", `{"url":"https://` + strings.Repeat("a", 2048) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler.Shorten(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestShortenStoreFailure(t *testing.T) {
	handler, mr := setupHandler(t)

	mr.Close()

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	rec := httptest.NewRecorder()

	handler.Shorten(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRedirectNotFound(t *testing.T) {
	handler, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	req.SetPathValue("code", "nope")
	rec := httptest.NewRecorder()

	handler.Redirect(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
