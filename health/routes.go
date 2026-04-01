// health provide plug and play route mapped at "GET {prefix}/health" for healthchecks
package health

import (
	"fmt"
	"net/http"

	"github.com/gato-preto-engenharia/pneuma"
)

type response struct {
	Status string `json:"status,omitempty"`
}

func Routes(prefix string) []pneuma.Route {
	return []pneuma.Route{
		{
			Name:    "pneuma.healthcheck",
			Pattern: fmt.Sprintf("GET %s/health", prefix),
			Handler: pneuma.Constantly(pneuma.Result{
				Status: http.StatusOK,
				Body: response{
					Status: "alive",
				},
			}),
		},
	}
}
