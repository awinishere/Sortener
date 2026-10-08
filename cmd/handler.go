package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
)

type Handler struct {
	store   *Store
	baseUrl string
}

type shortenRequest struct {
	Url string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortUrl string `json:"short_url"`
}

func validUrl(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (handler *Handler) Shorten(writer http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(writer, req.Body, 1024)

	var shortReq shortenRequest
	if err := json.NewDecoder(req.Body).Decode(&shortReq); err != nil {
		http.Error(writer, "Body must be JSON", http.StatusBadRequest)
		return
	}

	if !validUrl(shortReq.Url) {
		http.Error(writer, "Invalid URL", http.StatusBadRequest)
		return
	}

	code, err := handler.store.Save(req.Context(), shortReq.Url)
	if err != nil {
		http.Error(writer, "Failed to save URL", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(writer).Encode(shortenResponse{
		Code:     code,
		ShortUrl: handler.baseUrl + "/" + code,
	}); err != nil {
		log.Printf("failed to write shorten response: %v", err)
	}
}

func (handler *Handler) Redirect(writer http.ResponseWriter, req *http.Request) {
	target, err := handler.store.Get(req.Context(), req.PathValue("code"))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(writer, req)
		return
	}
	if err != nil {
		http.Error(writer, "there is an error", http.StatusInternalServerError)
		return
	}

	http.Redirect(writer, req, target, http.StatusFound)
}
