package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup pretty logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Parse command line args
	clusterContext := "kind-kind" // Default kind cluster
	if len(os.Args) > 1 {
		clusterContext = os.Args[1]
	}

	prometheusEndpoint := "" // Will try without Prometheus first
	if len(os.Args) > 2 {
		prometheusEndpoint = os.Args[2]
	}

	log.Info().Msg("🚀 HPA Watchdog - Collector Test")
	log.Info().Msg("================================")
	log.Info().Str("cluster_context", clusterContext).Msg("Target cluster")
	if prometheusEndpoint != "" {
		log.Info().Str("prometheus", prometheusEndpoint).Msg("Prometheus endpoint")
	} else {
		log.Info().Msg("Prometheus: disabled (using K8s metrics only)")
	}
	log.Info().Msg("")

	// Create cluster info
	cluster := &models.ClusterInfo{
		Name:    "test-cluster",
		Context: clusterContext,
		Server:  "https://127.0.0.1:6443", // Will be overridden by kubeconfig
	}

	// Create collector config
	config := monitor.DefaultCollectorConfig()
	config.ScanInterval = 10 * time.Second // Scan every 10s for testing
	config.ExcludeNamespaces = []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"monitoring",
		"logging",
		"default",
	}
	config.EnablePrometheus = prometheusEndpoint != ""

	// Create collector
	log.Info().Msg("Creating collector...")
	collector, err := monitor.NewCollector(cluster, prometheusEndpoint, config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create collector")
	}

	// Show collector stats
	stats := collector.GetStats()
	log.Info().
		Str("cluster", stats.Cluster).
		Bool("prometheus_enabled", stats.PrometheusEnabled).
		Bool("prometheus_connected", stats.PrometheusConnected).
		Msg("Collector initialized")

	// Create result channel
	resultChan := make(chan *monitor.ScanResult, 10)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutdown signal received, stopping...")
		cancel()
	}()

	// Start monitoring in background
	log.Info().Msg("")
	log.Info().Msg("Starting monitoring loop...")
	log.Info().Dur("interval", config.ScanInterval).Msg("Scan interval")
	log.Info().Msg("Press Ctrl+C to stop")
	log.Info().Msg("")

	go collector.StartMonitoring(ctx, resultChan)

	// Process results
	scanCount := 0
	totalSnapshots := 0
	totalAnomalies := 0

	for {
		select {
		case result := <-resultChan:
			scanCount++
			totalSnapshots += result.SnapshotsCount
			totalAnomalies += len(result.Anomalies)

			log.Info().Msg("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Info().
				Int("scan", scanCount).
				Str("cluster", result.Cluster).
				Time("timestamp", result.Timestamp).
				Msg("Scan completed")

			log.Info().
				Int("snapshots", result.SnapshotsCount).
				Int("anomalies", len(result.Anomalies)).
				Int("errors", len(result.Errors)).
				Dur("duration", result.Duration).
				Msg("Scan results")

			// Show errors if any
			if len(result.Errors) > 0 {
				log.Warn().Msg("Errors encountered:")
				for i, err := range result.Errors {
					log.Warn().Int("error", i+1).Err(err).Msg("")
				}
			}

			// Show anomalies
			if len(result.Anomalies) > 0 {
				log.Warn().Msg("🚨 Anomalies detected:")
				for i, anomaly := range result.Anomalies {
					log.Warn().
						Int("anomaly", i+1).
						Str("type", string(anomaly.Type)).
						Str("hpa", fmt.Sprintf("%s/%s", anomaly.Namespace, anomaly.HPAName)).
						Msg(anomaly.Message)

					// Show actions
					if len(anomaly.Actions) > 0 {
						log.Info().Msg("   Suggested actions:")
						for _, action := range anomaly.Actions {
							log.Info().Msgf("   - %s", action)
						}
					}
				}
			} else {
				log.Info().Msg("✅ No anomalies detected")
			}

			// Show cache stats
			cacheStats := collector.GetStats()
			log.Info().
				Int("total_hpas", cacheStats.TotalHPAs).
				Int("total_snapshots", cacheStats.TotalSnapshots).
				Int64("memory_bytes", cacheStats.MemoryUsage).
				Msgf("Cache stats (%.2f KB)", float64(cacheStats.MemoryUsage)/1024)

			// Summary
			log.Info().Msg("")
			log.Info().
				Int("total_scans", scanCount).
				Int("total_snapshots", totalSnapshots).
				Int("total_anomalies", totalAnomalies).
				Msg("Session summary")
			log.Info().Msg("")

		case <-ctx.Done():
			log.Info().Msg("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Info().Msg("Monitoring stopped")
			log.Info().Msg("")
			log.Info().Msg("Final Statistics:")
			log.Info().
				Int("total_scans", scanCount).
				Int("total_snapshots", totalSnapshots).
				Int("total_anomalies", totalAnomalies).
				Msg("Session totals")

			if scanCount > 0 {
				avgSnapshots := float64(totalSnapshots) / float64(scanCount)
				log.Info().
					Float64("avg_snapshots_per_scan", avgSnapshots).
					Msg("Averages")
			}

			log.Info().Msg("")
			log.Info().Msg("👋 Goodbye!")
			return
		}
	}
}
