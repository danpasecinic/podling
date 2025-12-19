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
	"github.com/danpasecinic/podling/internal/master/dns"
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

	dnsConfig := initDNS()
	dnsServer := dns.NewServer(store, dnsConfig)
	go func() {
		if err := dnsServer.Start(ctx); err != nil {
			log.Printf("DNS server error: %v", err)
		}
	}()

	server := api.NewServer(store, sched, endpointController)

	go server.StartNodeExpirationChecker(ctx)

	authConfig, authStore := initAuth(store)
	authMiddleware := auth.NewMiddleware(authConfig, authStore)
	authMiddleware.SetSkipPaths(
		"/health",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/signup",
	)
	authMiddleware.SetSkipPrefixes("/api/v1/nodes/")
	authMiddleware.SetSkipSuffixes("/status")

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(
		middleware.CORSWithConfig(
			middleware.CORSConfig{
				AllowOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
				AllowMethods: []string{
					http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions,
				},
				AllowHeaders: []string{
					echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization,
				},
			},
		),
	)

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
		if err := e.Start(":8070"); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	config.Enabled = true

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

	return config, authStore
}

func initDNS() dns.Config {
	config := dns.DefaultConfig()

	if enabled := os.Getenv("DNS_ENABLED"); enabled == "false" {
		config.Enabled = false
	}

	if port := os.Getenv("DNS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	if domain := os.Getenv("DNS_CLUSTER_DOMAIN"); domain != "" {
		config.ClusterDomain = domain
	}

	return config
}
