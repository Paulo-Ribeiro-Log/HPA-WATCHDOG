package portforward

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
)

// PortForward gerencia port-forward para Prometheus
type PortForward struct {
	cluster   string
	namespace string
	service   string
	localPort int
	cmd       *exec.Cmd
	cancel    context.CancelFunc
}

// Config configuração do port-forward
type Config struct {
	Cluster   string
	Namespace string // Default: "monitoring"
	Service   string // Default: "prometheus-k8s" ou "prometheus-server"
	LocalPort int    // Default: 9090
}

// New cria novo port-forward
func New(cfg Config) *PortForward {
	// Defaults
	if cfg.Namespace == "" {
		cfg.Namespace = "monitoring"
	}
	if cfg.Service == "" {
		cfg.Service = "prometheus-k8s"
	}
	if cfg.LocalPort == 0 {
		cfg.LocalPort = 9090
	}

	return &PortForward{
		cluster:   cfg.Cluster,
		namespace: cfg.Namespace,
		service:   cfg.Service,
		localPort: cfg.LocalPort,
	}
}

// Start inicia port-forward
func (pf *PortForward) Start() error {
	log.Info().
		Str("cluster", pf.cluster).
		Str("namespace", pf.namespace).
		Str("service", pf.service).
		Int("port", pf.localPort).
		Msg("Iniciando port-forward para Prometheus")

	ctx, cancel := context.WithCancel(context.Background())
	pf.cancel = cancel

	// Comando kubectl port-forward
	pf.cmd = exec.CommandContext(ctx,
		"kubectl",
		"port-forward",
		fmt.Sprintf("svc/%s", pf.service),
		fmt.Sprintf("%d:9090", pf.localPort),
		"-n", pf.namespace,
		"--context", pf.cluster,
	)

	// Inicia em background
	if err := pf.cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar port-forward: %w", err)
	}

	// Aguarda port-forward estar pronto
	if err := pf.waitForReady(); err != nil {
		pf.Stop()
		return err
	}

	log.Info().
		Str("cluster", pf.cluster).
		Int("port", pf.localPort).
		Msg("Port-forward ativo")

	return nil
}

// Stop para port-forward
func (pf *PortForward) Stop() error {
	if pf.cancel != nil {
		pf.cancel()
	}

	if pf.cmd != nil && pf.cmd.Process != nil {
		log.Info().
			Str("cluster", pf.cluster).
			Msg("Parando port-forward")

		if err := pf.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("falha ao parar port-forward: %w", err)
		}
	}

	return nil
}

// waitForReady aguarda port-forward estar pronto
func (pf *PortForward) waitForReady() error {
	url := fmt.Sprintf("http://localhost:%d/-/ready", pf.localPort)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout aguardando port-forward em %s", url)

		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// GetURL retorna URL do Prometheus
func (pf *PortForward) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", pf.localPort)
}

// IsRunning verifica se port-forward está ativo
func (pf *PortForward) IsRunning() bool {
	if pf.cmd == nil || pf.cmd.Process == nil {
		return false
	}

	// Tenta conectar
	url := fmt.Sprintf("http://localhost:%d/-/ready", pf.localPort)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// PortForwardManager gerencia múltiplos port-forwards
type PortForwardManager struct {
	forwards map[string]*PortForward
}

// NewManager cria novo gerenciador
func NewManager() *PortForwardManager {
	return &PortForwardManager{
		forwards: make(map[string]*PortForward),
	}
}

// Start inicia port-forward para um cluster
func (m *PortForwardManager) Start(cluster string) error {
	// Se já existe, retorna
	if pf, exists := m.forwards[cluster]; exists && pf.IsRunning() {
		log.Info().Str("cluster", cluster).Msg("Port-forward já ativo")
		return nil
	}

	// Cria novo port-forward (porta incrementada por cluster)
	basePort := 9090
	port := basePort + len(m.forwards)

	pf := New(Config{
		Cluster:   cluster,
		LocalPort: port,
	})

	if err := pf.Start(); err != nil {
		return err
	}

	m.forwards[cluster] = pf
	return nil
}

// Stop para port-forward de um cluster
func (m *PortForwardManager) Stop(cluster string) error {
	pf, exists := m.forwards[cluster]
	if !exists {
		return nil
	}

	if err := pf.Stop(); err != nil {
		return err
	}

	delete(m.forwards, cluster)
	return nil
}

// StopAll para todos os port-forwards
func (m *PortForwardManager) StopAll() error {
	for cluster, pf := range m.forwards {
		if err := pf.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("cluster", cluster).
				Msg("Erro ao parar port-forward")
		}
	}

	m.forwards = make(map[string]*PortForward)
	return nil
}

// GetURL retorna URL do Prometheus para um cluster
func (m *PortForwardManager) GetURL(cluster string) string {
	pf, exists := m.forwards[cluster]
	if !exists {
		return ""
	}
	return pf.GetURL()
}
