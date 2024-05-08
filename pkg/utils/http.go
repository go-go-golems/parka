package utils

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

type H map[string]interface{}

func WithPathPrefixMiddleware(path string, h echo.HandlerFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Response().Committed {
				return next(c)
			}

			if !strings.HasPrefix(c.Request().URL.Path, path) {
				return next(c)
			}

			err := h(c)
			if err != nil {
				if _, ok := err.(*NoPageFoundError); ok {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}

			return nil
		}
	}
}
