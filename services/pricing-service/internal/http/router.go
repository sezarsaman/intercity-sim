package http

import (
	stdhttp "net/http"

	chi "github.com/go-chi/chi/v5"
)

func NewRouter() stdhttp.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}
