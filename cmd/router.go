package main

import (
	"log/slog"
	"net/http"
	"time"

	_ "github.com/awn/sortener/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @BasePath annotation in main.go.
const apiPrefix = "/api/v1"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET "+apiPrefix+"/swagger/", httpSwagger.Handler(httpSwagger.URL(apiPrefix+"/swagger/doc.json")))
	mux.HandleFunc("POST "+apiPrefix+"/shorten", h.Shorten)
	mux.HandleFunc("GET "+apiPrefix+"/{code}", h.Redirect)

	return withLogging(mux)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(sw, req)
		slog.Info("request",
			"method", req.Method,
			"path", req.URL.Path,
			"status", sw.status,
			"dur", time.Since(start).Round(time.Microsecond),
			"remote", req.RemoteAddr,
		)
	})
}
