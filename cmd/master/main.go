package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/danpasecinic/podling/internal/auth"
	"github.com/danpasecinic/podling/internal/master/api"
	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/services"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
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

	endpointController := services.NewEndpointController(store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := endpointController.Start(ctx); err != nil {
			log.Printf("endpoint controller error: %v", err)
		}
	}()

	server := api.NewServer(store, sched, endpointController)

	go server.StartNodeExpirationChecker(ctx)

	authConfig, authStore := initAuth(store)
	authMiddleware := auth.NewMiddleware(authConfig, authStore)
	authMiddleware.SetSkipPaths("/health", "/api/v1/auth/login", "/api/v1/auth/refresh")

	if authConfig.Enabled {
		bootstrapAdminUser(authStore, authConfig)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET(
		"/health", func(c echo.Context) error {
			return c.JSON(
				http.StatusOK, map[string]string{
					"status":      "ok",
					"service":     "podling-master",
					"authEnabled": strconv.FormatBool(authConfig.Enabled),
				},
			)
		},
	)

	e.Use(authMiddleware.Authenticate())

	authHandlers := auth.NewAuthHandlers(
		authStore,
		authMiddleware.JWTManager(),
		authMiddleware.APIKeyManager(),
		authConfig,
	)
	authHandlers.RegisterRoutes(e, authMiddleware)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Info("server stopped")
}

func initStore() (*state.PostgresStateStore, func() error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Printf("initializing PostgreSQL store with connection: %s", maskPassword())
	pgStore, err := state.NewPostgresStateStore(dbURL)
	if err != nil {
		log.Fatalf("failed to initialize PostgreSQL store: %v", err)
	}

	log.Println("PostgreSQL store initialized successfully")
	return pgStore, pgStore.Close
}

func maskPassword() string {
	return "***masked***"
}

func initAuth(pgStore *state.PostgresStateStore) (auth.Config, auth.AuthStore) {
	config := auth.DefaultConfig()

	if enabled := os.Getenv("AUTH_ENABLED"); enabled == "true" || enabled == "1" {
		config.Enabled = true
	}

	config.JWTSecret = os.Getenv("JWT_SECRET")
	if config.JWTSecret == "" {
		config.JWTSecret = "podling-default-secret-change-in-production"
	}

	config.APIKeySecret = os.Getenv("API_KEY_SECRET")
	if config.APIKeySecret == "" {
		config.APIKeySecret = "podling-apikey-secret-change-in-production"
	}

	if expiry := os.Getenv("TOKEN_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			config.TokenExpiry = d
		}
	}

	authStore := auth.NewPostgresAuthStore(pgStore.DB())

	if config.Enabled {
		log.Println("authentication enabled")
	} else {
		log.Println("authentication disabled (set AUTH_ENABLED=true to enable)")
	}

	return config, authStore
}

func bootstrapAdminUser(authStore auth.AuthStore, config auth.Config) {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
		log.Println("WARNING: using default admin password. Set ADMIN_PASSWORD environment variable in production")
	}

	_, err := authStore.GetUserByUsername(username)
	if err == nil {
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("failed to hash admin password: %v", err)
		return
	}

	user := auth.User{
		ID:           "user-admin-bootstrap",
		Username:     username,
		PasswordHash: passwordHash,
		Role:         auth.RoleAdmin,
		CreatedAt:    time.Now(),
	}

	if err := authStore.AddUser(user); err != nil {
		log.Printf("failed to create admin user: %v", err)
		return
	}

	log.Printf("admin user '%s' created successfully", username)
}
