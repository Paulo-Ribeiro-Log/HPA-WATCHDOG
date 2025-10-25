package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Load carrega a configuração do arquivo YAML
func Load(configPath string) (*models.WatchdogConfig, error) {
	// Expande ~ para home directory
	if configPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[2:])
	}

	// Configura Viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Lê o arquivo
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	log.Info().Str("config", configPath).Msg("Configuration file loaded")

	// Parse para struct
	cfg := &models.WatchdogConfig{}

	// Monitoring
	cfg.ScanIntervalSeconds = viper.GetInt("monitoring.scan_interval_seconds")
	cfg.HistoryRetentionMinutes = viper.GetInt("monitoring.history_retention_minutes")

	// Prometheus
	cfg.PrometheusEnabled = viper.GetBool("monitoring.prometheus.enabled")
	cfg.PrometheusAutoDiscover = viper.GetBool("monitoring.prometheus.auto_discover")
	cfg.PrometheusFallback = viper.GetBool("monitoring.prometheus.fallback_to_metrics_server")
	cfg.PrometheusEndpoints = viper.GetStringMapString("monitoring.prometheus.endpoints")
	cfg.PrometheusDiscoveryPatterns = viper.GetStringSlice("monitoring.prometheus.discovery_patterns")

	// Alertmanager
	cfg.AlertmanagerEnabled = viper.GetBool("monitoring.alertmanager.enabled")
	cfg.AlertmanagerAutoDiscover = viper.GetBool("monitoring.alertmanager.auto_discover")
	cfg.AlertmanagerSyncInterval = viper.GetInt("monitoring.alertmanager.sync_interval_seconds")
	cfg.AlertmanagerEndpoints = viper.GetStringMapString("monitoring.alertmanager.endpoints")
	cfg.AlertmanagerDiscoveryPatterns = viper.GetStringSlice("monitoring.alertmanager.discovery_patterns")

	// Clusters
	cfg.ClustersConfigPath = viper.GetString("clusters.config_path")
	cfg.AutoDiscoverClusters = viper.GetBool("clusters.auto_discover")
	cfg.ExcludeClusters = viper.GetStringSlice("clusters.exclude")

	// Storage
	cfg.EnablePersistence = viper.GetBool("storage.enable_persistence")
	cfg.PersistencePath = viper.GetString("storage.persistence_path")

	// Alerts
	cfg.MaxActiveAlerts = viper.GetInt("alerts.max_active_alerts")
	cfg.AutoAckResolvedAlerts = viper.GetBool("alerts.auto_ack_resolved")
	cfg.SourcePriority = viper.GetStringSlice("alerts.source_priority")
	cfg.Deduplicate = viper.GetBool("alerts.deduplicate")
	cfg.DedupeWindowMinutes = viper.GetInt("alerts.dedupe_window_minutes")
	cfg.AutoCorrelate = viper.GetBool("alerts.auto_correlate")
	cfg.CorrelationWindowMinutes = viper.GetInt("alerts.correlation_window_minutes")

	// Thresholds
	cfg.Thresholds.ReplicaDeltaPercent = viper.GetFloat64("thresholds.replica_delta_percent")
	cfg.Thresholds.ReplicaDeltaAbsolute = int32(viper.GetInt("thresholds.replica_delta_absolute"))
	cfg.Thresholds.CPUWarningPercent = int32(viper.GetInt("thresholds.cpu_warning_percent"))
	cfg.Thresholds.CPUCriticalPercent = int32(viper.GetInt("thresholds.cpu_critical_percent"))
	cfg.Thresholds.MemoryWarningPercent = int32(viper.GetInt("thresholds.memory_warning_percent"))
	cfg.Thresholds.MemoryCriticalPercent = int32(viper.GetInt("thresholds.memory_critical_percent"))
	cfg.Thresholds.TargetDeviationPercent = viper.GetFloat64("thresholds.target_deviation_percent")
	cfg.Thresholds.ScalingStuckMinutes = viper.GetInt("thresholds.scaling_stuck_minutes")
	cfg.Thresholds.AlertOnConfigChange = viper.GetBool("thresholds.alert_on_config_change")
	cfg.Thresholds.AlertOnResourceChange = viper.GetBool("thresholds.alert_on_resource_change")
	cfg.Thresholds.RequestRateSpikePercent = viper.GetFloat64("thresholds.request_rate_spike_percent")
	cfg.Thresholds.ErrorRateCriticalPercent = viper.GetFloat64("thresholds.error_rate_critical_percent")
	cfg.Thresholds.P95LatencyCriticalMs = viper.GetFloat64("thresholds.p95_latency_critical_ms")

	// UI
	cfg.RefreshIntervalMs = viper.GetInt("ui.refresh_interval_ms")
	cfg.Theme = viper.GetString("ui.theme")
	cfg.EnableSounds = viper.GetBool("ui.enable_sounds")

	// Logging
	cfg.LogLevel = viper.GetString("logging.level")
	cfg.LogOutput = viper.GetString("logging.output")
	cfg.LogMaxSizeMB = viper.GetInt("logging.max_size_mb")
	cfg.LogMaxBackups = viper.GetInt("logging.max_backups")
	cfg.LogCompress = viper.GetBool("logging.compress")

	// Validação básica
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	log.Info().
		Int("scan_interval", cfg.ScanIntervalSeconds).
		Bool("prometheus", cfg.PrometheusEnabled).
		Bool("alertmanager", cfg.AlertmanagerEnabled).
		Msg("Configuration loaded successfully")

	return cfg, nil
}

// validate valida a configuração
func validate(cfg *models.WatchdogConfig) error {
	if cfg.ScanIntervalSeconds < 1 {
		return fmt.Errorf("scan_interval_seconds must be >= 1")
	}

	if cfg.HistoryRetentionMinutes < 1 {
		return fmt.Errorf("history_retention_minutes must be >= 1")
	}

	if cfg.MaxActiveAlerts < 1 {
		return fmt.Errorf("max_active_alerts must be >= 1")
	}

	// Valida thresholds
	if cfg.Thresholds.CPUWarningPercent < 1 || cfg.Thresholds.CPUWarningPercent > 100 {
		return fmt.Errorf("cpu_warning_percent must be between 1 and 100")
	}

	if cfg.Thresholds.CPUCriticalPercent < 1 || cfg.Thresholds.CPUCriticalPercent > 100 {
		return fmt.Errorf("cpu_critical_percent must be between 1 and 100")
	}

	if cfg.Thresholds.CPUCriticalPercent <= cfg.Thresholds.CPUWarningPercent {
		return fmt.Errorf("cpu_critical_percent must be > cpu_warning_percent")
	}

	if cfg.Thresholds.MemoryWarningPercent < 1 || cfg.Thresholds.MemoryWarningPercent > 100 {
		return fmt.Errorf("memory_warning_percent must be between 1 and 100")
	}

	if cfg.Thresholds.MemoryCriticalPercent < 1 || cfg.Thresholds.MemoryCriticalPercent > 100 {
		return fmt.Errorf("memory_critical_percent must be between 1 and 100")
	}

	if cfg.Thresholds.MemoryCriticalPercent <= cfg.Thresholds.MemoryWarningPercent {
		return fmt.Errorf("memory_critical_percent must be > memory_warning_percent")
	}

	return nil
}

// ExpandPath expande ~ para home directory em paths
func ExpandPath(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}
