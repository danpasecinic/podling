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
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", cfgFile)
	}

	// Check environment variable for master URL
	if envMaster := os.Getenv("PODLING_MASTER_URL"); envMaster != "" && masterURL == "http://localhost:8080" {
		masterURL = envMaster
	}
}

// GetMasterURL returns the configured master URL
func GetMasterURL() string {
	return masterURL
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}
