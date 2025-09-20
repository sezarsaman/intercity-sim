package http

import (
	"log"
	stdhttp "net/http"

	chi "github.com/go-chi/chi/v5"
)

func NewRouter() stdhttp.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("trip-service: write health response error: %v", err)
		}
	})

	return r
}
