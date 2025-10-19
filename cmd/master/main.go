package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	store := state.NewInMemoryStore()
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET(
		"/health", func(c echo.Context) error {
			return c.JSON(
				http.StatusOK, map[string]string{
					"status":  "ok",
					"service": "podling-master",
				},
			)
		},
	)

	v1 := e.Group("/api/v1")

	v1.GET(
		"/tasks", func(c echo.Context) error {
			tasks, err := store.ListTasks()
			if err != nil {
				return c.JSON(
					http.StatusInternalServerError, map[string]string{
						"error": err.Error(),
					},
				)
			}
			return c.JSON(http.StatusOK, tasks)
		},
	)

	v1.GET(
		"/nodes", func(c echo.Context) error {
			nodes, err := store.ListNodes()
			if err != nil {
				return c.JSON(
					http.StatusInternalServerError, map[string]string{
						"error": err.Error(),
					},
				)
			}
			return c.JSON(http.StatusOK, nodes)
		},
	)

	go func() {
		if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	e.Logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Info("server stopped")
}
