package main

import (
	"net/http"

	_ "github.com/awn/sortener/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const apiPrefix = "/api/v1"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET "+apiPrefix+"/swagger/", httpSwagger.Handler(httpSwagger.URL(apiPrefix+"/swagger/doc.json")))
	mux.HandleFunc("POST "+apiPrefix+"/shorten", h.Shorten)
	mux.HandleFunc("GET "+apiPrefix+"/{code}", h.Redirect)

	return mux
}
