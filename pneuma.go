// pneuma Provides a thin data-first wrapper over [net/http] server implementation
//
// for pneuma everything that can be treated as a value should be, such as setting
// response headers, registering routes, mapping results, etc...
package pneuma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
)

var (
	ErrCannotDecodeJson = errors.New("cannot decode as json")
)

// Headers is a wrapper over map\[string\]string
type Headers map[string]string

// Request Is pneuma abstraction over [http.Request]
type Request struct {
	// The request HTTP method
	Method string

	// The request headers
	Headers Headers

	// The request body byte array
	Body []byte

	// The underlaying [http.Request] object, note that the body will already be
	// closed when consuming it
	Raw *http.Request
}

func NewRequest(r *http.Request) Request {
	headers := make(Headers, len(r.Header))

	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}

	return Request{
		Method:  r.Method,
		Headers: headers,
		Body:    body,
		Raw:     r,
	}
}

// PathVariable Returns a path variable with the corresponding key
func (r Request) PathVariable(key string) (string, bool) {
	value := r.Raw.PathValue(key)
	if len(value) == 0 {
		return "", false
	}

	return value, true
}

// DecodeJson Decode the request bytes as a JSON into the target pointer using [json.Unmarshal]
func (r Request) DecodeJson(target any) error {
	if err := json.Unmarshal(r.Body, target); err != nil {
		return ErrCannotDecodeJson
	}

	return nil
}

// Result Is pneuma's response interface
type Result struct {
	// The result status code
	Status int

	// The result body, can be anything, including a [error]
	Body any

	// The result headers, can be nil
	Headers Headers
}

// Err Checks if the result body is a error and if so returns it
func (r Result) Err() error {
	if err, ok := r.Body.(error); ok {
		return err
	}

	return nil
}

// NewResult Construct a [Result] with status, body and a optional header, if multiple
// header maps are provided only the first will be used
func NewResult(status int, body any, headers ...Headers) Result {
	var resultHeaders Headers
	if len(headers) > 0 {
		resultHeaders = headers[0]
	}

	return Result{
		Status:  status,
		Body:    body,
		Headers: resultHeaders,
	}
}

// NewEmptyResult Construct a [Result] with only the status and a empty body, use
// [Result] struct directly if you want to provide more options
func NewEmptyResult(status int) Result {
	return NewResult(status, nil)
}

// Handler Is pneuma's request server-side handler
type Handler func(r Request) Result

// Middleware Is pneuma's server-side middleware implementation
type Middleware func(Handler) Handler

// Route Is pneuma's server-side routing implementation
type Route struct {
	// The route name, use only for tracing
	Name string `json:"name"`

	// The route pattern to match against
	Pattern string `json:"pattern"`

	// The route server-side handler
	Handler Handler `json:"-"`

	// The route specific middlewares, these will be applied after the server level
	// middlewares
	Middlewares []Middleware `json:"-"`
}

// NewRoute Provides a side-effect free constructor over [Route]
func NewRoute(pattern string, handler Handler, middlewares ...Middleware) Route {
	return Route{
		Pattern:     pattern,
		Handler:     handler,
		Middlewares: middlewares,
	}
}

// ServerSpec Is pneuma's basic HTTP server specification, it provides a side-effect free,
// data first approach to serving requests
type ServerSpec struct {
	// The server context, used for external termination, if not provided will use [context.Background]
	Ctx context.Context `json:"-"`

	// The address on which pneuma's internal [http.ServerMux] must listen to
	Address string `json:"address"`

	// The static headers that will be included on all responses, they are applied before
	// the [Handler] result headers
	Headers Headers `json:"headers"`

	// ResultEncoder is the function used to encode the response body, if not
	// provided pneuma will use the [JsonResultEncoder] function
	//
	// This can also be used to conditionally transform the [Result] body, one example
	// of this is how [JsonResultEncoder] handles non-nil errors
	//
	// Must return the response content-type followed by the encoded body as a byte slice
	ResultEncoder func(Result) (string, []byte) `json:"-"`

	// RecoverFunc is the function called when a handler panics, server-level panics
	// crash the application as expected, if not provided will log the error and
	// return 500
	RecoverFunc func(any) `json:"-"`

	// The middlewares that should be applied to every route in this spec, if present
	// they will be applied before the route specific middlewares
	Middlewares []Middleware `json:"-"`

	// The spec routes
	Routes []Route `json:"routes"`
}

// ListenAndServe Creates a [http.ServeMux] from a pneuma [ServerSpec] and starts listening on
// the provided address
//
// This uses the global [log/slog] for logging
func ListenAndServe(spec ServerSpec) error {
	mux := http.NewServeMux()

	slog.Debug("initialized net/http ServeMux")

	for _, route := range spec.Routes {
		if route.Handler == nil || len(route.Pattern) == 0 {
			continue
		}

		wrappedHandler := route.Handler
		for _, middleware := range slices.Backward(slices.Concat(spec.Middlewares, route.Middlewares)) {
			wrappedHandler = middleware(wrappedHandler)
		}

		encoderFn := JsonResultEncoder
		if spec.ResultEncoder != nil {
			encoderFn = spec.ResultEncoder
		}

		mux.HandleFunc(route.Pattern, func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {

					if spec.RecoverFunc != nil {
						spec.RecoverFunc(rec)
					} else {
						slog.Error("handler panicked", "error", rec)
						writer.WriteHeader(http.StatusInternalServerError)
					}

				}
			}()

			req := NewRequest(request)
			result := wrappedHandler(req)

			level := slog.LevelInfo
			if result.Status >= 400 && result.Status < 500 {
				level = slog.LevelWarn
			} else if result.Status >= 500 {
				level = slog.LevelError
			}

			slog.Log(
				req.Raw.Context(),
				level,
				"received request",
				slog.Group("req", "host", req.Raw.Host, "method", req.Raw.Method, "path", req.Raw.URL.Path),
				slog.Group("res", "status", result.Status))

			for headerKey, headerValue := range spec.Headers {
				writer.Header().Add(headerKey, headerValue)
			}

			for headerKey, headerValue := range result.Headers {
				writer.Header().Add(headerKey, headerValue)
			}

			if result.Body != nil {
				contentType, response := encoderFn(result)
				if contentType != "" && len(response) > 0 {
					writer.Header().Add("Content-Type", contentType)
					writer.WriteHeader(result.Status)

					if _, err := writer.Write(response); err != nil {
						slog.Error("failed to write response", "error", err.Error())
					}
				} else {
					slog.Warn("empty encoded response or content-type")
				}
			} else {
				writer.WriteHeader(result.Status)
			}
		})

		slog.Debug("registered Route", "pattern", route.Pattern, "name", route.Name)
	}

	server := &http.Server{Addr: spec.Address, Handler: mux}

	ctx := spec.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		<-ctx.Done()

		slog.Info("received stopping signal, shutting down the pneuma server")

		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown pneuma server", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	return nil
}

// MustListenAndServe Creates a [http.ServeMux] from a pneuma [ServerSpec] and starts listening on
// the provided address, panicking if it encounters any error
//
// This uses the global [log/slog] for logging
func MustListenAndServe(spec ServerSpec) {
	if err := ListenAndServe(spec); err != nil {
		panic(err)
	}
}

// Constantly returns a [Handler] that always responds with the given [Result],
// regardless of the request contents
func Constantly(res Result) Handler {
	return func(_ Request) Result {
		return res
	}
}

// JsonResultEncoder Is the default response mapper of pneuma server, it encodes
// the result as "application/json" and handles non-nil errors returned by [Result.Err]
func JsonResultEncoder(r Result) (string, []byte) {
	response := r.Body
	if err := r.Err(); err != nil {
		response = struct {
			Error string `json:"error,omitempty"`
		}{
			Error: err.Error(),
		}
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		encoded = []byte(`{"error": "failed to parse body as JSON"}`)
	}

	return "application/json", encoded
}
