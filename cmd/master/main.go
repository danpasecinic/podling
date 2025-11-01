package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danpasecinic/podling/internal/master/api"
	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	store, closer := initStore()
	if closer != nil {
		defer func() {
			if err := closer(); err != nil {
				log.Printf("error closing store: %v", err)
			}
		}()
	}

	sched := scheduler.NewRoundRobin()
	server := api.NewServer(store, sched)

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

	server.RegisterRoutes(e)

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

// initStore initializes the state store based on environment variables
// Returns the store and an optional closer function
func initStore() (state.StateStore, func() error) {
	storeType := os.Getenv("STORE_TYPE")
	if storeType == "" {
		storeType = "memory" // default to in-memory
	}

	switch storeType {
	case "postgres":
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			log.Fatal("DATABASE_URL environment variable is required when STORE_TYPE=postgres")
		}

		log.Printf("initializing PostgreSQL store with connection: %s", maskPassword())
		pgStore, err := state.NewPostgresStore(dbURL)
		if err != nil {
			log.Fatalf("failed to initialize PostgreSQL store: %v", err)
		}

		log.Println("PostgreSQL store initialized successfully")
		return pgStore, pgStore.Close

	case "memory":
		log.Println("using in-memory store (data will not persist)")
		return state.NewInMemoryStore(), nil

	default:
		log.Fatalf("unknown STORE_TYPE: %s (valid options: memory, postgres)", storeType)
		return nil, nil
	}
}

// maskPassword masks the password in a database URL for logging
func maskPassword() string {
	return "***masked***"
}
