package server

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/http/pprof"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	// Create a custom error response
	errorResponse := map[string]interface{}{
		"error": err.Error(),
		//"stackTrace": getStackTrace(),
		"query": c.QueryParams(),
		"url":   c.Request().URL.String(),
	}

	if err, ok := err.(stackTracer); ok {
		errorResponse["stackTrace"] = err.StackTrace()
	}

	log.Error().
		Err(err).
		Str("query", fmt.Sprintf("%v", c.QueryParams())).
		Str("url", c.Request().URL.String()).
		Msg("Error")

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD { // Issue #608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, errorResponse)
		}
	}

	_ = err
}

func (s *Server) RegisterDebugRoutes() {
	handlers_ := map[string]http.HandlerFunc{
		"/debug/pprof/":          pprof.Index,
		"/debug/pprof/cmdline":   pprof.Cmdline,
		"/debug/pprof/profile":   pprof.Profile,
		"/debug/pprof/symbol":    pprof.Symbol,
		"/debug/pprof/trace":     pprof.Trace,
		"/debug/pprof/mutex":     pprof.Index,
		"/debug/pprof/allocs":    pprof.Index,
		"/debug/pprof/block":     pprof.Index,
		"/debug/pprof/goroutine": pprof.Index,
		"/debug/pprof/heap":      pprof.Index,
	}

	for route, handler := range handlers_ {
		route_ := route
		handler_ := handler
		s.Router.GET(route_, echo.WrapHandler(handler_))
	}
}
