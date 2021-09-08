package config

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func New() *echo.Echo {
	InitEnvironmentVariables()
	echoInstance := echo.New()

	// Configuring Middleware Logger
	echoInstance.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		// Skipping logging for health checking api
		Skipper: func(c echo.Context) bool {
			if c.Request().RequestURI == "/health" {
				return true
			}
			return false
		},
		Format: "[${time_rfc3339}] method=${method}, uri=${uri}, status=${status}, latency=${latency_human}\n",
	}))

	echoInstance.Use(middleware.Recover())
	return echoInstance
}