package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DiscoverClusters descobre clusters do kubeconfig
func DiscoverClusters(cfg *models.WatchdogConfig) ([]models.ClusterInfo, error) {
	if !cfg.AutoDiscoverClusters {
		log.Info().Msg("Auto-discovery disabled, skipping cluster discovery")
		return []models.ClusterInfo{}, nil
	}

	// Path do kubeconfig (padrão: ~/.kube/kubeconfig)
	kubeconfigPath := getKubeconfigPath()

	log.Info().Str("path", kubeconfigPath).Msg("Discovering clusters from kubeconfig")

	// Carrega kubeconfig
	kubeconfig, err := loadKubeconfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Extrai clusters
	clusters := []models.ClusterInfo{}
	excludeMap := make(map[string]bool)
	for _, name := range cfg.ExcludeClusters {
		excludeMap[name] = true
	}

	for contextName, context := range kubeconfig.Contexts {
		clusterName := context.Cluster

		// Pula clusters excluídos
		if excludeMap[clusterName] {
			log.Debug().Str("cluster", clusterName).Msg("Cluster excluded from monitoring")
			continue
		}

		// Pega informações do cluster
		cluster, exists := kubeconfig.Clusters[clusterName]
		if !exists {
			log.Warn().
				Str("context", contextName).
				Str("cluster", clusterName).
				Msg("Cluster not found in kubeconfig, skipping")
			continue
		}

		// Determina se é o contexto default
		isDefault := contextName == kubeconfig.CurrentContext

		clusterInfo := models.ClusterInfo{
			Name:      clusterName,
			Context:   contextName,
			Server:    cluster.Server,
			Namespace: context.Namespace,
			IsDefault: isDefault,
			Status:    models.ClusterStatusOffline, // Será atualizado após verificação
		}

		clusters = append(clusters, clusterInfo)

		log.Debug().
			Str("cluster", clusterName).
			Str("context", contextName).
			Str("server", cluster.Server).
			Bool("default", isDefault).
			Msg("Cluster discovered")
	}

	log.Info().Int("count", len(clusters)).Msg("Clusters discovered")

	return clusters, nil
}

// getKubeconfigPath retorna o path do kubeconfig
// Prioridade: KUBECONFIG env var -> ~/.kube/kubeconfig -> ~/.kube/config
func getKubeconfigPath() string {
	// 1. KUBECONFIG env var
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		return envPath
	}

	// 2. Home directory
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get home directory, using default")
		return filepath.Join(".", ".kube", "kubeconfig")
	}

	// 3. Tenta ~/.kube/kubeconfig primeiro (seu caso)
	kubeconfigPath := filepath.Join(home, ".kube", "kubeconfig")
	if _, err := os.Stat(kubeconfigPath); err == nil {
		return kubeconfigPath
	}

	// 4. Fallback para ~/.kube/config (padrão kubectl)
	kubeconfigPath = filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); err == nil {
		return kubeconfigPath
	}

	// 5. Se nada existir, retorna o path do seu caso
	return filepath.Join(home, ".kube", "kubeconfig")
}

// loadKubeconfig carrega o arquivo kubeconfig
func loadKubeconfig(path string) (*api.Config, error) {
	// Verifica se arquivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s", path)
	}

	// Carrega usando client-go
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	if len(config.Clusters) == 0 {
		return nil, fmt.Errorf("no clusters found in kubeconfig")
	}

	return config, nil
}

// GetClusterConfig retorna a config de um cluster específico
func GetClusterConfig(clusterName string) (*api.Config, error) {
	kubeconfigPath := getKubeconfigPath()
	config, err := loadKubeconfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Procura o contexto para esse cluster
	for contextName, context := range config.Contexts {
		if context.Cluster == clusterName {
			// Cria config específica para esse contexto
			overrides := &clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			}

			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*config,
				contextName,
				overrides,
				nil,
			)

			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to create client config for cluster %s: %w", clusterName, err)
			}

			log.Debug().
				Str("cluster", clusterName).
				Str("context", contextName).
				Str("host", restConfig.Host).
				Msg("Cluster config created")

			return config, nil
		}
	}

	return nil, fmt.Errorf("cluster %s not found in kubeconfig", clusterName)
}

// ListClusters retorna apenas os nomes dos clusters (helper)
func ListClusters(cfg *models.WatchdogConfig) ([]string, error) {
	clusters, err := DiscoverClusters(cfg)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(clusters))
	for i, cluster := range clusters {
		names[i] = cluster.Name
	}

	return names, nil
}

// GetDefaultCluster retorna o cluster default (current-context)
func GetDefaultCluster() (string, error) {
	kubeconfigPath := getKubeconfigPath()
	config, err := loadKubeconfig(kubeconfigPath)
	if err != nil {
		return "", err
	}

	if config.CurrentContext == "" {
		return "", fmt.Errorf("no current-context set in kubeconfig")
	}

	context, exists := config.Contexts[config.CurrentContext]
	if !exists {
		return "", fmt.Errorf("current-context %s not found in kubeconfig", config.CurrentContext)
	}

	return context.Cluster, nil
}
