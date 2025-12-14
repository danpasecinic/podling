package dns

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/miekg/dns"
)

const (
	DefaultPort = 5353
)

type Config struct {
	Port          int
	ClusterDomain string
	Enabled       bool
}

func DefaultConfig() Config {
	return Config{
		Port:          DefaultPort,
		ClusterDomain: DefaultClusterDomain,
		Enabled:       true,
	}
}

type Server struct {
	config    Config
	store     state.StateStore
	handler   *Handler
	udpServer *dns.Server
	tcpServer *dns.Server
	mu        sync.RWMutex
	running   bool
}

func NewServer(store state.StateStore, config Config) *Server {
	handler := NewHandler(store, config.ClusterDomain)
	return &Server{
		config:  config,
		store:   store,
		handler: handler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		log.Println("DNS server is disabled")
		return nil
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("DNS server already running")
	}
	s.running = true
	s.mu.Unlock()

	addr := fmt.Sprintf(":%d", s.config.Port)

	dns.HandleFunc(".", s.handler.ServeDNS)

	s.udpServer = &dns.Server{
		Addr: addr,
		Net:  "udp",
	}

	s.tcpServer = &dns.Server{
		Addr: addr,
		Net:  "tcp",
	}

	errChan := make(chan error, 2)

	go func() {
		log.Printf("DNS server starting on %s (UDP)", addr)
		if err := s.udpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("UDP server error: %w", err)
		}
	}()

	go func() {
		log.Printf("DNS server starting on %s (TCP)", addr)
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("TCP server error: %w", err)
		}
	}()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	select {
	case err := <-errChan:
		s.Stop()
		return err
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	log.Println("DNS server stopping...")

	if s.udpServer != nil {
		if err := s.udpServer.Shutdown(); err != nil {
			log.Printf("Error shutting down UDP server: %v", err)
		}
	}

	if s.tcpServer != nil {
		if err := s.tcpServer.Shutdown(); err != nil {
			log.Printf("Error shutting down TCP server: %v", err)
		}
	}

	s.running = false
	log.Println("DNS server stopped")
}

func (s *Server) Address() string {
	return fmt.Sprintf(":%d", s.config.Port)
}

func (s *Server) Port() int {
	return s.config.Port
}
