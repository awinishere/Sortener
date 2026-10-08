package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// @title Sortener API
// @version 1.0.0
// @description URL shortener with a configurable alphabet and time-to-live, stored in Redis
// @BasePath /api/v1
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, falling back to system environment")
	}
	if err := LoadConfig(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	store := New(env("REDIS_ADDR", "localhost:6379"))
	h := &Handler{
		store:   store,
		baseUrl: env("BASE_URL", "http://localhost:8080"),
	}

	addr := env("PORT", ":8080")
	log.Println("running on", addr)
	log.Fatal(http.ListenAndServe(addr, NewRouter(h)))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
