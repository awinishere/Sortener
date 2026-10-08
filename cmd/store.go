package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	alphabet string
	ttl      time.Duration
)

func LoadConfig() error {
	alphabet = os.Getenv("ALPHABET")
	if alphabet == "" {
		return errors.New("ALPHABET is required")
	}

	v := os.Getenv("TTL")
	if v == "" {
		return errors.New("TTL is required")
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("invalid TTL %q: %w", v, err)
	}
	if d <= 0 {
		return fmt.Errorf("TTL must be positive, got %q", v)
	}
	ttl = d
	return nil
}

var ErrNotFound = errors.New("link not found")

type Store struct {
	redis *redis.Client
}

func New(addr string) *Store {
	return &Store{redis: redis.NewClient(&redis.Options{Addr: addr})}
}

// NewFromEnv builds a Store from REDIS_ADDR, REDIS_USERNAME and REDIS_PASSWORD.
// REDIS_ADDR accepts either a plain "host:port" or a redis:// / rediss:// URL
// (rediss:// enables TLS, as used by most managed/cloud Redis providers).
func NewFromEnv() (*Store, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, errors.New("REDIS_ADDR is required")
	}

	if strings.Contains(addr, "://") {
		opt, err := redis.ParseURL(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_ADDR %q: %w", addr, err)
		}
		// Credentials embedded in the URL win; otherwise fall back to env vars.
		if opt.Username == "" {
			opt.Username = os.Getenv("REDIS_USERNAME")
		}
		if opt.Password == "" {
			opt.Password = os.Getenv("REDIS_PASSWORD")
		}
		return &Store{redis: redis.NewClient(opt)}, nil
	}

	return &Store{redis: redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})}, nil
}

func encode(n int64) string {
	var b []byte
	for n > 0 {
		b = append(b, alphabet[n%int64(len(alphabet))])
		n /= int64(len(alphabet))
	}
	return string(b)
}

func (store *Store) Save(ctx context.Context, url string) (string, error) {
	n, err := store.redis.Incr(ctx, "counter").Result()
	if err != nil {
		return "", err
	}

	code := encode(n)
	if err := store.redis.Set(ctx, "link:"+code, url, ttl).Err(); err != nil {
		return "", err
	}
	return code, err
}

func (store *Store) Get(ctx context.Context, code string) (string, error) {
	url, err := store.redis.Get(ctx, "link:"+code).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return url, err
}

func (store *Store) NextID(ctx context.Context) (int64, error) {
	return store.redis.Incr(ctx, "counter").Result()
}

func (store *Store) Ping(ctx context.Context) error {
	return store.redis.Ping(ctx).Err()
}
