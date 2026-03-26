package handler

import (
	"errors"
	"net/http"

	"github.com/gato-preto-engenharia/pneuma"
)

func Info(r *http.Request) pneuma.Result {
	return pneuma.NewResult(
		http.StatusOK,
		map[string]string{
			"alive": "yes",
		},
		pneuma.Headers{
			"X-Header-1": "X-Value-1",
			"X-Header-2": "X-Value-2",
		},
	)
}

func Fail(r *http.Request) pneuma.Result {
	return pneuma.NewResult(
		http.StatusUnprocessableEntity, errors.New("failed to process request"))
}
