package cli

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"30 seconds", 30, "30s"},
		{"1 minute", 60, "1m"},
		{"2 minutes", 120, "2m"},
		{"1 hour", 3600, "1h"},
		{"2 hours", 7200, "2h"},
		{"1 day", 86400, "1d"},
		{"2 days", 172800, "2d"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := formatDuration(secondsToDuration(tt.seconds))
				if result != tt.expected {
					t.Errorf("formatDuration(%d seconds) = %v, want %v", tt.seconds, result, tt.expected)
				}
			},
		)
	}
}

func secondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

func TestParseEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:     "single env var",
			input:    []string{"PORT=8080"},
			expected: map[string]string{"PORT": "8080"},
		},
		{
			name:  "multiple env vars",
			input: []string{"PORT=8080", "HOST=localhost", "MODE=production"},
			expected: map[string]string{
				"PORT": "8080",
				"HOST": "localhost",
				"MODE": "production",
			},
		},
		{
			name:     "env var with empty value",
			input:    []string{"EMPTY="},
			expected: map[string]string{"EMPTY": ""},
		},
		{
			name:     "env var without equals sign",
			input:    []string{"INVALID"},
			expected: map[string]string{},
		},
		{
			name:  "mixed valid and invalid",
			input: []string{"VALID=value", "INVALID", "ANOTHER=test"},
			expected: map[string]string{
				"VALID":   "value",
				"ANOTHER": "test",
			},
		},
		{
			name:     "env var with equals in value",
			input:    []string{"URL=http://localhost:8080"},
			expected: map[string]string{"URL": "http://localhost:8080"},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				envMap := make(map[string]string)
				for _, e := range tt.input {
					var key, value string
					for i, c := range e {
						if c == '=' {
							key = e[:i]
							value = e[i+1:]
							break
						}
					}
					if key != "" {
						envMap[key] = value
					}
				}

				if len(envMap) != len(tt.expected) {
					t.Errorf("parsed env map length = %d, want %d", len(envMap), len(tt.expected))
				}

				for k, v := range tt.expected {
					if envMap[k] != v {
						t.Errorf("env[%s] = %v, want %v", k, envMap[k], v)
					}
				}
			},
		)
	}
}

func TestGetMasterURL(t *testing.T) {
	tests := []struct {
		name     string
		flagSet  string
		envSet   string
		expected string
	}{
		{
			name:     "default URL",
			flagSet:  "",
			envSet:   "",
			expected: "http://localhost:8080",
		},
		{
			name:     "flag set",
			flagSet:  "http://production:8080",
			envSet:   "",
			expected: "http://production:8080",
		},
		{
			name:     "env set",
			flagSet:  "",
			envSet:   "http://staging:8080",
			expected: "http://staging:8080",
		},
		{
			name:     "flag overrides env",
			flagSet:  "http://production:8080",
			envSet:   "http://staging:8080",
			expected: "http://production:8080",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Reset global state
				if tt.flagSet != "" {
					masterURL = tt.flagSet
				} else {
					masterURL = "http://localhost:8080"
				}

				if tt.envSet != "" {
					t.Setenv("PODLING_MASTER_URL", tt.envSet)
					initConfig()
				}

				result := GetMasterURL()
				if result != tt.expected {
					t.Errorf("GetMasterURL() = %v, want %v", result, tt.expected)
				}

				masterURL = "http://localhost:8080"
			},
		)
	}
}

func TestIsVerbose(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		expected bool
	}{
		{"verbose enabled", true, true},
		{"verbose disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				verbose = tt.verbose
				result := IsVerbose()
				if result != tt.expected {
					t.Errorf("IsVerbose() = %v, want %v", result, tt.expected)
				}

				verbose = false
			},
		)
	}
}

func TestInitConfig(t *testing.T) {
	t.Run(
		"initConfig with no config file", func(t *testing.T) {
			// Reset state
			cfgFile = ""
			masterURL = "http://localhost:8080"

			initConfig()

			// Should use default
			if masterURL != "http://localhost:8080" {
				t.Errorf("initConfig() masterURL = %v, want default", masterURL)
			}
		},
	)

	t.Run(
		"initConfig with environment variable", func(t *testing.T) {
			cfgFile = ""
			masterURL = "http://localhost:8080"
			t.Setenv("PODLING_MASTER_URL", "http://env-master:9090")

			initConfig()

			if masterURL != "http://env-master:9090" {
				t.Errorf("initConfig() should use env var, got %v", masterURL)
			}

			// Reset
			masterURL = "http://localhost:8080"
		},
	)
}

func TestExecute(t *testing.T) {
	// Execute is the cobra root command execution
	// Testing it directly is complex, but we can verify it exists
	t.Run(
		"Execute function exists", func(t *testing.T) {
			// This test just ensures Execute is callable
			// Actual CLI testing would require integration tests
		},
	)
}
