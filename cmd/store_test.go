package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestLoadConfig(t *testing.T) {
	origAlphabet, origTTL := alphabet, ttl
	defer func() { alphabet, ttl = origAlphabet, origTTL }()

	t.Run("reads values from env", func(t *testing.T) {
		t.Setenv("ALPHABET", "abc")
		t.Setenv("TTL", "720h")

		if err := LoadConfig(); err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if alphabet != "abc" {
			t.Errorf("alphabet = %q, want %q", alphabet, "abc")
		}
		if ttl != 720*time.Hour {
			t.Errorf("ttl = %v, want %v", ttl, 720*time.Hour)
		}
	})

	t.Run("supports various duration formats", func(t *testing.T) {
		t.Setenv("ALPHABET", "xy")
		tests := []struct {
			in   string
			want time.Duration
		}{
			{"12h", 12 * time.Hour},
			{"30m", 30 * time.Minute},
			{"1h30m", 90 * time.Minute},
		}
		for _, tt := range tests {
			t.Setenv("TTL", tt.in)
			if err := LoadConfig(); err != nil {
				t.Fatalf("LoadConfig() with TTL=%q error = %v", tt.in, err)
			}
			if ttl != tt.want {
				t.Errorf("TTL %q = %v, want %v", tt.in, ttl, tt.want)
			}
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name     string
			alphabet string
			ttl      string
		}{
			{"missing alphabet", "", "1h"},
			{"missing ttl", "abc", ""},
			{"invalid ttl", "abc", "abc"},
			{"ttl without unit", "abc", "720"},
			{"zero ttl", "abc", "0s"},
			{"negative ttl", "abc", "-1h"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Setenv("ALPHABET", tt.alphabet)
				t.Setenv("TTL", tt.ttl)

				if err := LoadConfig(); err == nil {
					t.Errorf("LoadConfig() with ALPHABET=%q TTL=%q = nil, want error", tt.alphabet, tt.ttl)
				}
			})
		}
	})
}

func TestEncode(t *testing.T) {
	orig := alphabet
	defer func() { alphabet = orig }()

	t.Run("base10 alphabet", func(t *testing.T) {
		alphabet = "0123456789"
		tests := []struct {
			n    int64
			want string
		}{
			{0, ""},
			{1, "1"},
			{9, "9"},
			{10, "01"},
			{123, "321"},
		}
		for _, tt := range tests {
			if got := encode(tt.n); got != tt.want {
				t.Errorf("encode(%d) = %q, want %q", tt.n, got, tt.want)
			}
		}
	})

	t.Run("custom alphabet reverses digit order", func(t *testing.T) {
		alphabet = "abc"
		if got := encode(3); got != "ab" {
			t.Errorf("encode(3) = %q, want %q", got, "ab")
		}
	})
}

func setupStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()

	origAlphabet, origTTL := alphabet, ttl
	t.Cleanup(func() { alphabet, ttl = origAlphabet, origTTL })

	alphabet = "0123456789"
	ttl = time.Hour

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	return New(mr.Addr()), mr
}

func TestSaveAndGet(t *testing.T) {
	store, mr := setupStore(t)
	ctx := context.Background()

	code, err := store.Save(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if code == "" {
		t.Fatal("Save() returned empty code")
	}

	got, err := mr.Get("link:" + code)
	if err != nil {
		t.Fatalf("miniredis Get() error = %v", err)
	}
	if got != "https://example.com" {
		t.Errorf("redis value = %q, want %q", got, "https://example.com")
	}

	if gotTTL := mr.TTL("link:" + code); gotTTL != ttl {
		t.Errorf("redis TTL = %v, want %v", gotTTL, ttl)
	}

	got, err = store.Get(ctx, code)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "https://example.com" {
		t.Errorf("Get() = %q, want %q", got, "https://example.com")
	}
}

func TestSaveIncrementsCounter(t *testing.T) {
	store, _ := setupStore(t)
	ctx := context.Background()

	first, err := store.Save(ctx, "https://a.com")
	if err != nil {
		t.Fatalf("Save() #1 error = %v", err)
	}
	second, err := store.Save(ctx, "https://b.com")
	if err != nil {
		t.Fatalf("Save() #2 error = %v", err)
	}

	if first == second {
		t.Errorf("Save() returned duplicate code %q for two calls", first)
	}
}

func TestGetNotFound(t *testing.T) {
	store, _ := setupStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestGetExpired(t *testing.T) {
	store, mr := setupStore(t)
	ctx := context.Background()

	code, err := store.Save(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	mr.FastForward(ttl + time.Second)

	if _, err := store.Get(ctx, code); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after expiry error = %v, want ErrNotFound", err)
	}
}
