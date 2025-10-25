package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "absolute path",
			input:    "/etc/config.yaml",
			expected: "/etc/config.yaml",
		},
		{
			name:     "tilde path",
			input:    "~/.kube/config",
			expected: filepath.Join(home, ".kube/config"),
		},
		{
			name:     "relative path",
			input:    "./config.yaml",
			expected: "./config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if err != nil {
				t.Fatalf("ExpandPath() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("ExpandPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Cria arquivo de config temporário
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "watchdog.yaml")

	configContent := `
monitoring:
  scan_interval_seconds: 30
  history_retention_minutes: 5
  prometheus:
    enabled: true
    auto_discover: true
    fallback_to_metrics_server: true
  alertmanager:
    enabled: true
    auto_discover: true
    sync_interval_seconds: 30

clusters:
  auto_discover: true
  exclude:
    - test-cluster

storage:
  enable_persistence: false

alerts:
  max_active_alerts: 100
  auto_ack_resolved: true
  source_priority:
    - alertmanager
    - watchdog
  deduplicate: true
  dedupe_window_minutes: 5
  auto_correlate: true
  correlation_window_minutes: 10

thresholds:
  replica_delta_percent: 50.0
  replica_delta_absolute: 5
  cpu_warning_percent: 85
  cpu_critical_percent: 90
  memory_warning_percent: 85
  memory_critical_percent: 90
  target_deviation_percent: 30.0
  scaling_stuck_minutes: 10
  alert_on_config_change: true
  alert_on_resource_change: true
  request_rate_spike_percent: 100.0
  error_rate_critical_percent: 5.0
  p95_latency_critical_ms: 1000

ui:
  refresh_interval_ms: 500
  theme: "dark"
  enable_sounds: false

logging:
  level: "info"
  output: "/tmp/watchdog.log"
  max_size_mb: 100
  max_backups: 3
  compress: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Testa load
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Valida valores
	if cfg.ScanIntervalSeconds != 30 {
		t.Errorf("ScanIntervalSeconds = %d, want 30", cfg.ScanIntervalSeconds)
	}

	if cfg.Thresholds.CPUWarningPercent != 85 {
		t.Errorf("CPUWarningPercent = %d, want 85", cfg.Thresholds.CPUWarningPercent)
	}

	if cfg.Thresholds.CPUCriticalPercent != 90 {
		t.Errorf("CPUCriticalPercent = %d, want 90", cfg.Thresholds.CPUCriticalPercent)
	}

	if !cfg.PrometheusEnabled {
		t.Error("PrometheusEnabled = false, want true")
	}

	if !cfg.AutoDiscoverClusters {
		t.Error("AutoDiscoverClusters = false, want true")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T) string
		wantErr   bool
	}{
		{
			name: "invalid scan interval",
			configure: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "watchdog.yaml")
				content := `
monitoring:
  scan_interval_seconds: 0
thresholds:
  cpu_warning_percent: 85
  cpu_critical_percent: 90
  memory_warning_percent: 85
  memory_critical_percent: 90
alerts:
  max_active_alerts: 100
`
				os.WriteFile(configPath, []byte(content), 0644)
				return configPath
			},
			wantErr: true,
		},
		{
			name: "cpu_critical <= cpu_warning",
			configure: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "watchdog.yaml")
				content := `
monitoring:
  scan_interval_seconds: 30
thresholds:
  cpu_warning_percent: 90
  cpu_critical_percent: 85
  memory_warning_percent: 85
  memory_critical_percent: 90
alerts:
  max_active_alerts: 100
`
				os.WriteFile(configPath, []byte(content), 0644)
				return configPath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.configure(t)
			_, err := Load(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
