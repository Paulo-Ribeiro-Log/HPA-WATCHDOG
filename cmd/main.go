package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/config"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// CLI flags
	cfgFile string
	debug   bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hpa-watchdog",
	Short: "HPA Watchdog - Monitor autônomo para Kubernetes HPAs",
	Long: `HPA Watchdog é um monitor autônomo para Horizontal Pod Autoscalers (HPAs)
em múltiplos clusters Kubernetes.

Features:
  • Monitoramento multi-cluster em tempo real
  • Integração com Prometheus e Alertmanager
  • Interface TUI interativa (Bubble Tea)
  • Detecção de anomalias e alertas
  • Análise temporal e correlação`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🐕 HPA Watchdog starting...")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Config: %s\n", cfgFile)
		fmt.Printf("Debug: %v\n", debug)
		fmt.Println()
		fmt.Println("⚠️  TUI não implementado ainda. Use 'make run' após implementação.")
		// TODO: Inicializar TUI e monitoring loops
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostra informações de versão",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("HPA Watchdog\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Commit:     %s\n", Commit)
		fmt.Printf("Build Time: %s\n", BuildTime)
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Valida o arquivo de configuração",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Validating config file: %s\n", cfgFile)

		// Carrega config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Configuration is invalid: %v\n", err)
			os.Exit(1)
		}

		// Mostra resumo
		fmt.Println("✅ Configuration is valid")
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf("  Scan Interval: %ds\n", cfg.ScanIntervalSeconds)
		fmt.Printf("  History Retention: %dm\n", cfg.HistoryRetentionMinutes)
		fmt.Printf("  Prometheus: %v\n", cfg.PrometheusEnabled)
		fmt.Printf("  Alertmanager: %v\n", cfg.AlertmanagerEnabled)
		fmt.Printf("  Auto-discover Clusters: %v\n", cfg.AutoDiscoverClusters)
		fmt.Printf("  Max Active Alerts: %d\n", cfg.MaxActiveAlerts)
	},
}

var clustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "Lista clusters descobertos",
	Long:  "Lista todos os clusters descobertos do kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup logging
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		// Carrega config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Descobre clusters
		clusters, err := config.DiscoverClusters(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to discover clusters: %v\n", err)
			os.Exit(1)
		}

		if len(clusters) == 0 {
			fmt.Println("⚠️  No clusters found in kubeconfig")
			return
		}

		// Mostra clusters
		fmt.Printf("📊 Found %d cluster(s):\n\n", len(clusters))
		for i, cluster := range clusters {
			defaultMark := ""
			if cluster.IsDefault {
				defaultMark = " (default)"
			}
			fmt.Printf("%d. %s%s\n", i+1, cluster.Name, defaultMark)
			fmt.Printf("   Context:   %s\n", cluster.Context)
			fmt.Printf("   Server:    %s\n", cluster.Server)
			if cluster.Namespace != "" {
				fmt.Printf("   Namespace: %s\n", cluster.Namespace)
			}
			fmt.Println()
		}
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exporta histórico de alertas",
	Long:  "Exporta histórico de alertas em formato JSON ou CSV",
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		fmt.Printf("Exporting alerts to %s (format: %s)\n", output, format)
		// TODO: Implementar export
		fmt.Println("⚠️  Export não implementado ainda")
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Testa conexão e coleta de métricas de um HPA específico",
	Long: `Testa conexão ao cluster e coleta dados de um HPA específico.

Exemplos:
  # Testar todos HPAs de um namespace
  hpa-watchdog test --cluster production --namespace default

  # Testar HPA específico
  hpa-watchdog test --cluster production --namespace default --hpa my-app

  # Com métricas do Prometheus
  hpa-watchdog test --cluster production --namespace default --prometheus

  # Mostrar histórico de 5 minutos
  hpa-watchdog test --cluster production --namespace default --history

Via variáveis de ambiente:
  export TEST_CLUSTER_CONTEXT=production
  export TEST_NAMESPACE=default
  export TEST_HPA_NAME=my-app
  export COLLECT_METRICS=true
  export SHOW_HISTORY=true
  hpa-watchdog test`,
	Run: func(cmd *cobra.Command, args []string) {
		// Setup logging
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		// Pega flags
		cluster, _ := cmd.Flags().GetString("cluster")
		namespace, _ := cmd.Flags().GetString("namespace")
		hpaName, _ := cmd.Flags().GetString("hpa")
		collectPrometheus, _ := cmd.Flags().GetBool("prometheus")
		history, _ := cmd.Flags().GetBool("history")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Valida
		if cluster == "" {
			fmt.Fprintf(os.Stderr, "❌ --cluster é obrigatório\n")
			os.Exit(1)
		}

		if namespace == "" {
			fmt.Fprintf(os.Stderr, "❌ --namespace é obrigatório\n")
			os.Exit(1)
		}

		// Executa teste integrado
		if err := runIntegratedTest(cluster, namespace, hpaName, collectPrometheus, history, verbose || debug); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Teste falhou: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✅ Teste concluído com sucesso!")
	},
}

// runIntegratedTest executa teste integrado K8s + Prometheus
func runIntegratedTest(cluster, namespace, hpaName string, collectPrometheus, showHistory, verbose bool) error {
	ctx := context.Background()

	fmt.Printf("🧪 HPA Watchdog - Teste Integrado\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("   Cluster:    %s\n", cluster)
	fmt.Printf("   Namespace:  %s\n", namespace)
	if hpaName != "" {
		fmt.Printf("   HPA:        %s\n", hpaName)
	}
	if collectPrometheus {
		fmt.Printf("   Prometheus: ✅ habilitado\n")
	}
	if showHistory {
		fmt.Printf("   Histórico:  ✅ 5 minutos\n")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 1. Setup K8s Client
	log.Info().Msg("🔌 Conectando ao cluster...")
	clusterInfo := &models.ClusterInfo{
		Name:    cluster,
		Context: cluster,
	}

	k8sClient, err := monitor.NewK8sClient(clusterInfo)
	if err != nil {
		return fmt.Errorf("falha ao criar K8s client: %w", err)
	}

	// 2. Test Connection
	log.Info().Msg("🔍 Testando conexão...")
	if err := k8sClient.TestConnection(ctx); err != nil {
		return fmt.Errorf("falha ao conectar ao cluster: %w", err)
	}
	fmt.Println("✅ Cluster conectado")
	fmt.Println()

	// 3. List HPAs
	log.Info().Str("namespace", namespace).Msg("📊 Listando HPAs...")
	hpas, err := k8sClient.ListHPAs(ctx, namespace)
	if err != nil {
		return fmt.Errorf("falha ao listar HPAs: %w", err)
	}

	if len(hpas) == 0 {
		fmt.Printf("⚠️  Nenhum HPA encontrado no namespace '%s'\n", namespace)
		return nil
	}

	fmt.Printf("✅ %d HPA(s) encontrado(s)\n", len(hpas))
	fmt.Println()

	// 4. Filter specific HPA if requested
	type hpaType = autoscalingv2.HorizontalPodAutoscaler
	var targetHPAs []hpaType
	if hpaName != "" {
		found := false
		for _, hpa := range hpas {
			if hpa.Name == hpaName {
				targetHPAs = append(targetHPAs, hpa)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("HPA '%s' não encontrado no namespace '%s'", hpaName, namespace)
		}
	} else {
		targetHPAs = hpas
	}

	// 5. Setup Prometheus (if enabled)
	var promClient *prometheus.Client
	var promHealth *models.PrometheusHealth
	var pfMgr *monitor.PortForwardManager

	if collectPrometheus {
		log.Info().Msg("🔍 Descobrindo Prometheus...")

		// Try auto-discovery
		promClient, promHealth, err = prometheus.DiscoverAndConnect(
			ctx,
			k8sClient.Clientset,
			cluster,
			"monitoring",
		)

		if err != nil {
			log.Warn().Err(err).Msg("⚠️  Prometheus não disponível, continuando sem métricas")
			fmt.Println("⚠️  Prometheus não encontrado (continuando apenas com dados do K8s)")
			fmt.Println()
		} else {
			fmt.Printf("✅ Prometheus conectado\n")
			fmt.Printf("   Endpoint: %s\n", promHealth.Endpoint)
			fmt.Printf("   Version:  %s\n", promHealth.Version)
			fmt.Printf("   Targets:  %d ativos / %d dropped\n", promHealth.ActiveTargets, promHealth.DroppedTargets)
			fmt.Println()
		}
	}

	// Cleanup port-forward on exit
	if pfMgr != nil {
		defer pfMgr.Shutdown()
	}

	// 6. Collect data from each HPA
	for i := range targetHPAs {
		hpa := &targetHPAs[i]

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("📊 HPA %d/%d\n", i+1, len(targetHPAs))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Println()

		// Collect K8s snapshot
		snapshot, err := k8sClient.CollectHPASnapshot(ctx, hpa)
		if err != nil {
			log.Error().Err(err).Msg("Falha ao coletar snapshot")
			continue
		}

		// Enrich with Prometheus if available
		if promClient != nil {
			log.Info().Msg("📈 Coletando métricas do Prometheus...")
			if err := promClient.EnrichSnapshot(ctx, snapshot); err != nil {
				log.Warn().Err(err).Msg("⚠️  Falha ao coletar algumas métricas do Prometheus")
			} else {
				snapshot.DataSource = models.DataSourcePrometheus
				fmt.Println("✅ Métricas do Prometheus coletadas")
			}
		}

		// Print snapshot
		printDetailedSnapshot(snapshot, showHistory)
		fmt.Println()
	}

	return nil
}

// printDetailedSnapshot imprime snapshot detalhado
func printDetailedSnapshot(s *models.HPASnapshot, showHistory bool) {
	fmt.Printf("📍 Nome: %s/%s\n", s.Namespace, s.Name)
	fmt.Printf("🕐 Timestamp: %s\n", s.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Config
	fmt.Println("⚙️  Configuração:")
	fmt.Printf("   Min/Max Replicas:  %d / %d\n", s.MinReplicas, s.MaxReplicas)
	if s.CPUTarget > 0 {
		fmt.Printf("   CPU Target:        %d%%\n", s.CPUTarget)
	}
	if s.MemoryTarget > 0 {
		fmt.Printf("   Memory Target:     %d%%\n", s.MemoryTarget)
	}
	fmt.Println()

	// Status
	fmt.Println("📊 Status Atual:")
	fmt.Printf("   Current Replicas:  %d\n", s.CurrentReplicas)
	fmt.Printf("   Desired Replicas:  %d\n", s.DesiredReplicas)
	fmt.Printf("   Ready:             %v\n", s.Ready)
	fmt.Printf("   Scaling Active:    %v\n", s.ScalingActive)
	if s.LastScaleTime != nil {
		ago := time.Since(*s.LastScaleTime)
		fmt.Printf("   Last Scale:        %s (%s ago)\n",
			s.LastScaleTime.Format("2006-01-02 15:04:05"),
			formatDuration(ago))
	}
	fmt.Println()

	// Resources
	fmt.Println("💾 Resources (por pod):")
	if s.CPURequest != "" {
		fmt.Printf("   CPU Request:       %s\n", s.CPURequest)
	}
	if s.CPULimit != "" {
		fmt.Printf("   CPU Limit:         %s\n", s.CPULimit)
	}
	if s.MemoryRequest != "" {
		fmt.Printf("   Memory Request:    %s\n", s.MemoryRequest)
	}
	if s.MemoryLimit != "" {
		fmt.Printf("   Memory Limit:      %s\n", s.MemoryLimit)
	}
	fmt.Println()

	// Metrics
	if s.DataSource == models.DataSourcePrometheus || s.CPUCurrent > 0 || s.MemoryCurrent > 0 {
		fmt.Printf("📈 Métricas (%s):\n", s.DataSource)
		if s.CPUCurrent >= 0 {
			fmt.Printf("   CPU Atual:         %.2f%%", s.CPUCurrent)
			if s.CPUTarget > 0 {
				deviation := s.CPUCurrent - float64(s.CPUTarget)
				fmt.Printf(" (target: %d%%, desvio: %+.2f%%)", s.CPUTarget, deviation)
			}
			fmt.Println()
		}
		if s.MemoryCurrent >= 0 {
			fmt.Printf("   Memory Atual:      %.2f%%", s.MemoryCurrent)
			if s.MemoryTarget > 0 {
				deviation := s.MemoryCurrent - float64(s.MemoryTarget)
				fmt.Printf(" (target: %d%%, desvio: %+.2f%%)", s.MemoryTarget, deviation)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Extended metrics
	if s.RequestRate > 0 || s.ErrorRate > 0 || s.P95Latency > 0 {
		fmt.Println("🌐 Métricas Estendidas:")
		if s.RequestRate > 0 {
			fmt.Printf("   Request Rate:      %.2f req/s\n", s.RequestRate)
		}
		if s.ErrorRate >= 0 {
			fmt.Printf("   Error Rate:        %.2f%%\n", s.ErrorRate)
		}
		if s.P95Latency > 0 {
			fmt.Printf("   P95 Latency:       %.2f ms\n", s.P95Latency)
		}
		if s.NetworkRxBytes > 0 {
			fmt.Printf("   Network RX:        %.2f KB/s\n", s.NetworkRxBytes/1024)
		}
		if s.NetworkTxBytes > 0 {
			fmt.Printf("   Network TX:        %.2f KB/s\n", s.NetworkTxBytes/1024)
		}
		fmt.Println()
	}

	// History
	if showHistory {
		if len(s.CPUHistory) > 0 {
			fmt.Println("📊 Histórico CPU (5 min):")
			for i, val := range s.CPUHistory {
				fmt.Printf("   T-%ds: %.2f%%\n", (len(s.CPUHistory)-i)*30, val)
			}
			fmt.Println()
		}
		if len(s.MemoryHistory) > 0 {
			fmt.Println("📊 Histórico Memory (5 min):")
			for i, val := range s.MemoryHistory {
				fmt.Printf("   T-%ds: %.2f%%\n", (len(s.MemoryHistory)-i)*30, val)
			}
			fmt.Println()
		}
		if len(s.ReplicaHistory) > 0 {
			fmt.Println("📊 Histórico Replicas (5 min):")
			for i, val := range s.ReplicaHistory {
				fmt.Printf("   T-%ds: %d\n", (len(s.ReplicaHistory)-i)*30, val)
			}
			fmt.Println()
		}
	}

	// Quick anomaly analysis
	fmt.Println("🔍 Análise Rápida:")
	anomalies := detectQuickAnomalies(s)
	if len(anomalies) == 0 {
		fmt.Println("   ✅ Nenhuma anomalia detectada")
	} else {
		for _, anomaly := range anomalies {
			fmt.Printf("   %s %s\n", anomaly.icon, anomaly.message)
		}
	}
}

type quickAnomaly struct {
	icon    string
	message string
}

func detectQuickAnomalies(s *models.HPASnapshot) []quickAnomaly {
	var anomalies []quickAnomaly

	// 1. Maxed out
	if s.CurrentReplicas >= s.MaxReplicas && s.CPUCurrent > float64(s.CPUTarget)+20 {
		anomalies = append(anomalies, quickAnomaly{
			icon:    "🔴",
			message: fmt.Sprintf("MAXED OUT: no limite (%d) com CPU %.2f%% (target: %d%%)",
				s.MaxReplicas, s.CPUCurrent, s.CPUTarget),
		})
	}

	// 2. Underutilization
	if s.CurrentReplicas > 3 && s.CPUTarget > 0 && s.CPUCurrent < float64(s.CPUTarget)-40 {
		anomalies = append(anomalies, quickAnomaly{
			icon:    "🟡",
			message: fmt.Sprintf("UNDERUTILIZED: CPU %.2f%% muito abaixo do target %d%%",
				s.CPUCurrent, s.CPUTarget),
		})
	}

	// 3. Missing memory target
	if s.MemoryTarget == 0 {
		anomalies = append(anomalies, quickAnomaly{
			icon:    "🟡",
			message: "CONFIG: Memory target não configurado",
		})
	}

	// 4. High error rate
	if s.ErrorRate > 5.0 {
		anomalies = append(anomalies, quickAnomaly{
			icon:    "🔴",
			message: fmt.Sprintf("HIGH ERROR RATE: %.2f%% (crítico >5%%)", s.ErrorRate),
		})
	}

	// 5. High latency
	if s.P95Latency > 1000 {
		anomalies = append(anomalies, quickAnomaly{
			icon:    "🔴",
			message: fmt.Sprintf("HIGH LATENCY: P95 %.2fms (>1000ms)", s.P95Latency),
		})
	}

	// 6. Oscillation
	if len(s.ReplicaHistory) >= 5 {
		changes := 0
		for i := 1; i < len(s.ReplicaHistory); i++ {
			if s.ReplicaHistory[i] != s.ReplicaHistory[i-1] {
				changes++
			}
		}
		if changes > 3 {
			anomalies = append(anomalies, quickAnomaly{
				icon:    "🔴",
				message: fmt.Sprintf("OSCILLATION: %d mudanças de réplicas em 5min", changes),
			})
		}
	}

	return anomalies
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%dd", days)
	}
	months := days / 30
	return fmt.Sprintf("%dmo", months)
}

func init() {
	// Root command flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "configs/watchdog.yaml", "arquivo de configuração")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "habilita modo debug (logs verbosos)")

	// Export command flags
	exportCmd.Flags().StringP("output", "o", "alerts.json", "arquivo de saída")
	exportCmd.Flags().StringP("format", "f", "json", "formato de exportação (json, csv)")

	// Test command flags
	testCmd.Flags().StringP("cluster", "c", "", "cluster context (obrigatório)")
	testCmd.Flags().StringP("namespace", "n", "", "namespace (obrigatório)")
	testCmd.Flags().String("hpa", "", "nome do HPA (opcional, testa todos se vazio)")
	testCmd.Flags().BoolP("prometheus", "p", false, "coletar métricas do Prometheus")
	testCmd.Flags().Bool("history", false, "mostrar histórico de 5 minutos")
	testCmd.Flags().BoolP("verbose", "v", false, "logs verbosos")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(clustersCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(testCmd)
}
