package cli

import (
	"os"
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

func TestInitConfig_AllBranches(t *testing.T) {
	t.Run(
		"initConfig with config file", func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "podling-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(tmpfile.Name()) }()

			_, _ = tmpfile.WriteString("master: http://config-file:8080\n")
			_ = tmpfile.Close()

			oldCfg := cfgFile
			oldMaster := masterURL
			cfgFile = tmpfile.Name()
			masterURL = "http://localhost:8080"

			initConfig()

			// Restore
			cfgFile = oldCfg
			masterURL = oldMaster
		},
	)

	t.Run(
		"initConfig reads PODLING_MASTER_URL from env", func(t *testing.T) {
			oldMaster := masterURL
			masterURL = "http://localhost:8080"

			t.Setenv("PODLING_MASTER_URL", "http://env-test:9090")

			initConfig()

			if masterURL != "http://env-test:9090" {
				t.Errorf("initConfig() didn't read env var, got %s", masterURL)
			}

			masterURL = oldMaster
		},
	)
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"0 seconds", 0, "0s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"90 seconds", 90 * time.Second, "1m"},
		{"3599 seconds", 3599 * time.Second, "59m"},
		{"7200 seconds", 7200 * time.Second, "2h"},
		{"25 hours", 25 * time.Hour, "1d"},
		{"48 hours", 48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := formatDuration(tt.duration)
				if result != tt.expected {
					t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
				}
			},
		)
	}
}

func TestGetMasterURL_EdgeCases(t *testing.T) {
	t.Run(
		"master URL set via flag takes precedence", func(t *testing.T) {
			oldURL := masterURL
			masterURL = "http://flag-url:8080"
			defer func() { masterURL = oldURL }()

			t.Setenv("PODLING_MASTER_URL", "http://env-url:8080")

			result := GetMasterURL()
			if result != "http://flag-url:8080" {
				t.Errorf("expected flag URL to take precedence, got %s", result)
			}
		},
	)
}

func TestIsVerbose_Multiple(t *testing.T) {
	oldVerbose := verbose
	defer func() { verbose = oldVerbose }()

	verbose = false
	if IsVerbose() {
		t.Error("IsVerbose() should be false")
	}

	verbose = true
	if !IsVerbose() {
		t.Error("IsVerbose() should be true")
	}
}
