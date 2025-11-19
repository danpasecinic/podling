package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danpasecinic/podling/internal/worker/agent"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	nodeID := flag.String("node-id", "", "Node ID (required)")
	hostname := flag.String("hostname", "localhost", "Worker hostname")
	port := flag.Int("port", 8081, "Worker port")
	masterURL := flag.String("master-url", "http://localhost:8070", "Master API URL")
	heartbeatInterval := flag.Duration("heartbeat-interval", 30*time.Second, "Heartbeat interval")
	shutdownTimeout := flag.Duration("shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")

	flag.Parse()
	if *nodeID == "" {
		log.Fatal("node-id is required")
	}

	workerAgent, err := agent.NewAgent(*nodeID, *masterURL)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}
	defer workerAgent.Stop()

	log.Printf("registering worker with master at %s", *masterURL)
	if err := workerAgent.Register(*hostname, *port); err != nil {
		log.Fatalf("failed to register with master: %v", err)
	}

	workerAgent.Start(*heartbeatInterval)

	server := agent.NewServer(*nodeID, *hostname, *port, workerAgent)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET(
		"/health", func(c echo.Context) error {
			return c.JSON(
				http.StatusOK, map[string]string{
					"status":  "ok",
					"service": "podling-worker",
					"nodeId":  *nodeID,
				},
			)
		},
	)

	server.RegisterRoutes(e)

	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("worker starting on %s (node: %s)", addr, *nodeID)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("shutdown signal received, beginning graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()

	log.Println("stopping agent and waiting for running tasks...")
	if err := workerAgent.Shutdown(ctx); err != nil {
		log.Printf("warning: agent shutdown error: %v", err)
	}

	log.Println("shutting down HTTP server...")
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v", err)
	}

	log.Println("worker stopped gracefully")
}
