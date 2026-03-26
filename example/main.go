package main

import (
	"log/slog"
	"net/http"
	"os"
	"slices"

	"github.com/gato-preto-engenharia/pneuma"
	"github.com/gato-preto-engenharia/pneuma/example/encoder"
	"github.com/gato-preto-engenharia/pneuma/example/handler"
	"github.com/gato-preto-engenharia/pneuma/example/routing"
	"github.com/gato-preto-engenharia/pneuma/health"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	spec := pneuma.ServerSpec{
		Address:       ":3000",
		ResultEncoder: encoder.Result,
		Headers: map[string]string{
			"X-App-Name":    "Pneuma-Example",
			"X-App-Version": "v1.0.1",
		},
		Routes: slices.Concat(
			health.Routes,
			routing.External,
			[]pneuma.Route{
				pneuma.NewRoute("GET /info", handler.Info),
				pneuma.NewRoute("GET /fail", handler.Fail),
			},
		),
		RecoverFunc: func(r any) {
			slog.Error("Recovered!", "reason", r)
		},
		Middlewares: []pneuma.Middleware{
			func(next pneuma.Handler) pneuma.Handler {
				return func(r *http.Request) pneuma.Result {
					slog.Info("Spec level middleware!")

					return next(r)
				}
			},
		},
	}

	slog.Info("Starting pneuma server")

	if err := pneuma.ListenAndServe(spec); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
