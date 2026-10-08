package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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
