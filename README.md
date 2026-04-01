# Pneuma

A thin wrapper and utility library over Go net/http

## Getting started

### Example usage

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gato-preto-engenharia/pneuma"
)

func main() error {
	spec := pneuma.ServerSpec{
		Address: ":8000",
		Routes: []pneuma.Route{
			pneuma.NewRoute("GET /status", func(_ pneuma.Request) pneuma.Result {
				return pneuma.NewResult(http.StatusOK, map[string]string{
					"status": "alive",
				})
			}),
		},
	}

	if err := pneuma.ListenAndServe(spec); err != nil {
		panic(err)
	}
}
```

A more detailed example can be viewed at [example/](https://github.com/gato-preto-engenharia/pneuma/tree/main/example)

