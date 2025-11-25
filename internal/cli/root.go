package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile   string
	masterURL string
	verbose   bool
	apiKey    string
	token     string
)

var rootCmd = &cobra.Command{
	Use:   "podling",
	Short: "Podling - A lightweight container orchestrator",
	Long: `Podling is a lightweight container orchestrator built from scratch in Go.

It features a master controller with REST API, worker agents that manage containers 
via Docker, and this CLI tool for interacting with the system.`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.podling.yaml)")
	rootCmd.PersistentFlags().StringVar(&masterURL, "master", "http://localhost:8080", "master API URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "JWT token for authentication")
}

func initConfig() {
	if cfgFile != "" {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", cfgFile)
	}

	if envMaster := os.Getenv("PODLING_MASTER_URL"); envMaster != "" && masterURL == "http://localhost:8080" {
		masterURL = envMaster
	}

	if apiKey == "" {
		apiKey = os.Getenv("PODLING_API_KEY")
	}

	if token == "" {
		token = os.Getenv("PODLING_TOKEN")
	}
}

func GetMasterURL() string {
	return masterURL
}

func IsVerbose() bool {
	return verbose
}

func GetAPIKey() string {
	return apiKey
}

func GetToken() string {
	return token
}

func NewAuthenticatedClient() *Client {
	client := NewClient(masterURL)
	if token != "" {
		client.SetToken(token)
	} else if apiKey != "" {
		client.SetAPIKey(apiKey)
	}
	return client
}
