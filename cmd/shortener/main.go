package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"mime"
	"net/http"
	"strings"
)

type Storage map[string]string

func (s Storage) Get(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

func (s Storage) Put(key, value string) {
	s[key] = value
}

func (s Storage) KeyByValue(value string) (string, bool) {
	for k, v := range s {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (s Storage) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// NOTE: to mock randomHex in tests
var randomHexImpl = func(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	if err := http.ListenAndServe(`:8080`, ShortenURLRouter()); err != nil {
		panic(err)
	}
}

func ShortenURLRouter() chi.Router {
	router := chi.NewRouter()
	storage := make(Storage)
	router.Post("/", CreateShortenedURLHandler(storage))
	router.Get("/{id}", GetShortenedURLHandler(storage))

	return router
}

func GetShortenedURLHandler(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET accepted", http.StatusBadRequest)
			return
		}
		if r.URL.Path == `/` {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		pathSplitted := strings.Split(r.URL.Path, `/`)
		if len(pathSplitted) != 2 {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		shortenedPath := pathSplitted[len(pathSplitted)-1]
		originalURL, ok := storage.KeyByValue(shortenedPath)
		if !ok {
			http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
			return
		}
		http.RedirectHandler(originalURL, http.StatusTemporaryRedirect).ServeHTTP(w, r)
	}
}

func CreateShortenedURLHandler(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST accepted", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if !hasContentType(r, "text/plain") {
			http.Error(w, `Only "text/plain" accepted`, http.StatusBadRequest)
			return
		}

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		url := string(bytes)

		shortenedURL, ok := storage.Get(url)
		if !ok {
			path, err := randomHex(8)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			shortenedURL = fmt.Sprintf("http://localhost:8080/%v", path)
			storage.Put(url, path)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortenedURL))
	}
}

func randomHex(n int) (string, error) {
	return randomHexImpl(n)
}

func hasContentType(r *http.Request, mimetype string) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return mimetype == "application/octet-stream"
	}

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == mimetype {
			return true
		}
	}
	return false
}
