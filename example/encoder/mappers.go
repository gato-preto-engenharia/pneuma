package encoder

import (
	"encoding/json"
	"time"

	"github.com/gato-preto-engenharia/pneuma"
)

type Response struct {
	TS      int64 `json:"ts,omitempty"`
	IsError bool  `json:"is_error,omitempty"`
	Data    any   `json:"data,omitempty"`
}

type Error struct {
	Code string `json:"code,omitempty"`
	Err  string `json:"err,omitempty"`
}

func Result(r pneuma.Result) (string, []byte) {
	isError := false
	data := r.Body
	if err := r.Err(); err != nil {
		isError = true
		data = Error{
			Code: "internal.error",
			Err:  err.Error(),
		}
	}

	encodedBody, _ := json.Marshal(Response{
		TS:      time.Now().UnixMilli(),
		IsError: isError,
		Data:    data,
	})

	return "application/json", encodedBody
}
