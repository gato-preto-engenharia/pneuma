package routing

import (
	"log/slog"
	"net/http"

	"github.com/gato-preto-engenharia/pneuma"
)

var External = []pneuma.Route{
	{
		// Directly using the struct to use all properties instead of only the
		// ones accessible by the util constructors
		Name:    "routing.external.constantly",
		Pattern: "GET /constantly",
		Handler: pneuma.Constantly(pneuma.NewEmptyResult(http.StatusNoContent)),
		Middlewares: []pneuma.Middleware{
			func(next pneuma.Handler) pneuma.Handler {
				return func(r *http.Request) pneuma.Result {
					slog.Info("Route level middleware!")

					return next(r)
				}
			},
		},
	},
}
