package app

import (
	"context"
	stdhttp "net/http"
)

type App struct {
	srv *stdhttp.Server
}

func New(port string, handler stdhttp.Handler) *App {
	return &App{
		srv: &stdhttp.Server{
			Addr:    ":" + port,
			Handler: handler,
		},
	}
}

func (a *App) Start(ctx context.Context) error {
	return a.srv.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	return a.srv.Shutdown(ctx)
}
