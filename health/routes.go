// health provide plug and play route mapped at "GET /pneuma/v1/health" for healthchecks
package health

import (
	"net/http"

	"github.com/gato-preto-engenharia/pneuma"
)

type Response struct {
	Status string `json:"status,omitempty"`
}

var Routes = []pneuma.Route{
	{
		Name:    "pneuma.healthcheck",
		Pattern: "GET /pneuma/v1/health",
		Handler: pneuma.Constantly(pneuma.Result{
			Status: http.StatusOK,
			Body: Response{
				Status: "alive",
			},
		}),
	},
}
