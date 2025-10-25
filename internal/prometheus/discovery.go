package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
	"github.com/rs/zerolog/log"
)

// DiscoveryConfig configuração para auto-discovery
type DiscoveryConfig struct {
	Enabled          bool
	Namespaces       []string // namespaces onde procurar (default: monitoring)
	ServicePatterns  []string // padrões de nome de serviço
	UsePortForward   bool     // usar port-forward manager
	LocalPort        int      // porta local para port-forward
	PortForwardMgr   *monitor.PortForwardManager
}

// DefaultDiscoveryConfig retorna configuração padrão
func DefaultDiscoveryConfig() *DiscoveryConfig {
	return &DiscoveryConfig{
		Enabled: true,
		Namespaces: []string{
			"monitoring",
			"prometheus",
			"kube-prometheus",
		},
		ServicePatterns: []string{
			"prometheus",
			"prometheus-server",
			"prometheus-operated",
			"kube-prometheus-stack-prometheus",
			"prometheus-k8s",
		},
		UsePortForward: true,
		LocalPort:      monitor.DefaultLocalPort,
	}
}

// DiscoverPrometheus tenta descobrir endpoint do Prometheus no cluster
func DiscoverPrometheus(ctx context.Context, k8sClient *monitor.K8sClient, config *DiscoveryConfig) (string, error) {
	if config == nil {
		config = DefaultDiscoveryConfig()
	}

	log.Info().
		Str("cluster", k8sClient.GetClusterInfo().Name).
		Msg("Starting Prometheus discovery")

	// Para cada namespace
	for _, namespace := range config.Namespaces {
		// Para cada padrão de serviço
		for _, pattern := range config.ServicePatterns {
			endpoint, err := tryDiscoverService(ctx, k8sClient, namespace, pattern, config)
			if err != nil {
				log.Debug().
					Err(err).
					Str("namespace", namespace).
					Str("pattern", pattern).
					Msg("Service not found")
				continue
			}

			log.Info().
				Str("cluster", k8sClient.GetClusterInfo().Name).
				Str("namespace", namespace).
				Str("service", pattern).
				Str("endpoint", endpoint).
				Msg("Prometheus discovered")

			return endpoint, nil
		}
	}

	return "", fmt.Errorf("prometheus not found in cluster %s", k8sClient.GetClusterInfo().Name)
}

// tryDiscoverService tenta descobrir um serviço específico
func tryDiscoverService(ctx context.Context, k8sClient *monitor.K8sClient, namespace, serviceName string, config *DiscoveryConfig) (string, error) {
	// Obtém informações do serviço via reflection (hack para acessar clientset)
	// Precisaríamos adicionar um método GetClientset() ao K8sClient

	// Por enquanto, vamos assumir que o serviço existe e usar port-forward
	if config.UsePortForward && config.PortForwardMgr != nil {
		// Inicia port-forward
		if err := config.PortForwardMgr.StartPortForward(
			k8sClient.GetClusterInfo().Name,
			namespace,
			serviceName,
			9090, // porta padrão Prometheus
		); err != nil {
			return "", err
		}

		endpoint := fmt.Sprintf("http://localhost:%d", config.LocalPort)

		// Verifica se Prometheus está respondendo
		promClient, err := NewClient(k8sClient.GetClusterInfo().Name, endpoint)
		if err != nil {
			return "", err
		}

		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := promClient.TestConnection(testCtx); err != nil {
			return "", fmt.Errorf("prometheus not responding: %w", err)
		}

		log.Info().
			Str("cluster", k8sClient.GetClusterInfo().Name).
			Str("namespace", namespace).
			Str("service", serviceName).
			Str("endpoint", endpoint).
			Msg("Port-forward to Prometheus established")

		return endpoint, nil
	}

	// Fallback: ClusterIP direto (dentro do cluster)
	endpoint := fmt.Sprintf("http://%s.%s.svc:9090", serviceName, namespace)
	return endpoint, nil
}

// DiscoverAllPrometheusEndpoints descobre endpoints em múltiplos clusters
func DiscoverAllPrometheusEndpoints(
	ctx context.Context,
	k8sClients map[string]*monitor.K8sClient,
	config *DiscoveryConfig,
) map[string]string {
	endpoints := make(map[string]string)

	for clusterName, client := range k8sClients {
		endpoint, err := DiscoverPrometheus(ctx, client, config)
		if err != nil {
			log.Warn().
				Err(err).
				Str("cluster", clusterName).
				Msg("Failed to discover Prometheus")
			continue
		}

		endpoints[clusterName] = endpoint
	}

	log.Info().
		Int("clusters_total", len(k8sClients)).
		Int("prometheus_found", len(endpoints)).
		Msg("Prometheus discovery complete")

	return endpoints
}

// VerifyPrometheusEndpoint verifica se um endpoint está funcional
func VerifyPrometheusEndpoint(ctx context.Context, cluster, endpoint string) error {
	client, err := NewClient(cluster, endpoint)
	if err != nil {
		return err
	}

	return client.TestConnection(ctx)
}

// GetPrometheusVersion obtém a versão do Prometheus
func GetPrometheusVersion(ctx context.Context, client *Client) (string, error) {
	buildInfo, err := client.api.Buildinfo(ctx)
	if err != nil {
		return "", err
	}

	version := buildInfo.Version
	if version == "" {
		return "unknown", nil
	}

	return version, nil
}

// GetPrometheusConfig obtém configuração do Prometheus
func GetPrometheusConfig(ctx context.Context, client *Client) (map[string]interface{}, error) {
	config, err := client.api.Config(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"yaml": config.YAML,
	}, nil
}

// GetPrometheusTargets obtém targets configurados no Prometheus
func GetPrometheusTargets(ctx context.Context, client *Client) (map[string]interface{}, error) {
	targets, err := client.api.Targets(ctx)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	droppedCount := 0

	if targets.Active != nil {
		activeCount = len(targets.Active)
	}
	if targets.Dropped != nil {
		droppedCount = len(targets.Dropped)
	}

	return map[string]interface{}{
		"active_count":  activeCount,
		"dropped_count": droppedCount,
		"active":        targets.Active,
		"dropped":       targets.Dropped,
	}, nil
}

// CheckPrometheusHealth verifica saúde do Prometheus
func CheckPrometheusHealth(ctx context.Context, endpoint string) (*models.PrometheusHealth, error) {
	client, err := NewClient("health-check", endpoint)
	if err != nil {
		return nil, err
	}

	health := &models.PrometheusHealth{
		Endpoint:  endpoint,
		Timestamp: time.Now(),
	}

	// Testa conexão
	if err := client.TestConnection(ctx); err != nil {
		health.Healthy = false
		health.Error = err.Error()
		return health, nil
	}

	health.Healthy = true
	health.Connected = true

	// Obtém versão
	if version, err := GetPrometheusVersion(ctx, client); err == nil {
		health.Version = version
	}

	// Obtém targets
	if targets, err := GetPrometheusTargets(ctx, client); err == nil {
		health.ActiveTargets = targets["active_count"].(int)
		health.DroppedTargets = targets["dropped_count"].(int)
	}

	log.Info().
		Str("endpoint", endpoint).
		Bool("healthy", health.Healthy).
		Str("version", health.Version).
		Int("targets", health.ActiveTargets).
		Msg("Prometheus health check complete")

	return health, nil
}

// DiscoverAndConnect descobre e conecta ao Prometheus em um cluster
// Retorna o client, health info e erro
func DiscoverAndConnect(ctx context.Context, clientset interface{}, cluster, namespace string) (*Client, *models.PrometheusHealth, error) {
	// Por hora, vamos tentar endpoints conhecidos diretamente
	// TODO: Implementar discovery real via K8s API quando resolver import cycle

	knownServices := []string{
		"prometheus-k8s-prometheus-1",
		"prometheus-server",
		"prometheus-operated",
		"kube-prometheus-stack-prometheus",
		"prometheus-k8s",
		"prometheus",
	}

	for _, serviceName := range knownServices {
		// Tenta endpoint ClusterIP
		endpoint := fmt.Sprintf("http://%s.%s.svc:9090", serviceName, namespace)

		client, err := NewClient(cluster, endpoint)
		if err != nil {
			continue
		}

		// Testa conexão
		testCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		if err := client.TestConnection(testCtx); err != nil {
			continue
		}

		// Coleta health info
		health, err := CheckPrometheusHealth(ctx, endpoint)
		if err != nil || !health.Healthy {
			continue
		}

		log.Info().
			Str("cluster", cluster).
			Str("namespace", namespace).
			Str("service", serviceName).
			Str("endpoint", endpoint).
			Msg("✅ Prometheus discovered and connected")

		return client, health, nil
	}

	return nil, nil, fmt.Errorf("prometheus not found in namespace %s", namespace)
}
