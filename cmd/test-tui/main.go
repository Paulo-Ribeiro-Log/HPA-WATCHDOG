package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuração de logging (para arquivo, não interfere na TUI)
	logFile, err := os.OpenFile("/tmp/hpa-watchdog-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Logger = zerolog.New(logFile).With().Timestamp().Logger()
	}

	log.Info().Msg("Iniciando HPA Watchdog TUI (test mode)")

	// Cria model da TUI
	model := tui.New()

	// Inicia programa Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Goroutine para gerar dados de teste
	go generateTestData(model)

	// Executa TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao executar TUI: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("HPA Watchdog TUI finalizado")
}

func generateTestData(m tui.Model) {
	log.Info().Msg("Gerando dados de teste...")

	// Simula dados de 3 clusters
	clusters := []string{
		"akspriv-faturamento-prd-admin",
		"akspriv-payment-prd-admin",
		"akspriv-api-prd-admin",
	}

	namespaces := []string{"ingress-nginx", "istio-system", "default", "kube-system"}
	hpaNames := []string{"nginx-controller", "istio-gateway", "api-deployment", "web-frontend"}

	// Gera snapshots iniciais
	for _, cluster := range clusters {
		for i, namespace := range namespaces {
			if i > 2 { // Limite de HPAs por cluster
				break
			}

			snapshot := &models.HPASnapshot{
				Timestamp:       time.Now(),
				Cluster:         cluster,
				Namespace:       namespace,
				Name:            hpaNames[i%len(hpaNames)],
				MinReplicas:     2,
				MaxReplicas:     10,
				CurrentReplicas: 3 + int32(i),
				DesiredReplicas: 3 + int32(i),
				CPUTarget:       70,
				CPUCurrent:      50.0 + float64(i*10),
				MemoryCurrent:   60.0,
				Ready:           true,
				ScalingActive:   true,
				DataSource:      models.DataSourcePrometheus,
			}

			// Envia snapshot para TUI
			select {
			case m.GetSnapshotChan() <- snapshot:
				log.Debug().Str("hpa", snapshot.Name).Msg("Snapshot enviado")
			default:
				log.Warn().Msg("Canal de snapshots cheio")
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	// Aguarda um pouco e gera anomalias de teste
	time.Sleep(2 * time.Second)

	// Anomalia 1: OSCILLATION
	anomaly1 := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeOscillation,
		Severity:  models.SeverityWarning,
		Cluster:   clusters[0],
		Namespace: namespaces[0],
		HPAName:   hpaNames[0],
		Timestamp: time.Now(),
		Message:   "HPA oscilando: 7 mudanças de réplicas em 5m0s. Indica instabilidade no comportamento de scaling.",
		Actions: []string{
			"Aumentar stabilizationWindowSeconds no HPA",
			"Revisar métricas de CPU/Memory - podem estar oscilando",
			"Verificar se há job/cron impactando carga",
		},
		Snapshot: &models.HPASnapshot{
			CurrentReplicas: 5,
			MinReplicas:     2,
			MaxReplicas:     10,
			CPUCurrent:      75.0,
			MemoryCurrent:   60.0,
		},
	}

	select {
	case m.GetAnomalyChan() <- anomaly1:
		log.Info().Msg("Anomalia OSCILLATION enviada")
	default:
		log.Warn().Msg("Canal de anomalias cheio")
	}

	time.Sleep(1 * time.Second)

	// Anomalia 2: CPU_SPIKE
	anomaly2 := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeCPUSpike,
		Severity:  models.SeverityWarning,
		Cluster:   clusters[1],
		Namespace: namespaces[1],
		HPAName:   hpaNames[1],
		Timestamp: time.Now(),
		Message:   "CPU spike: 45.0% → 95.0% (+111.1% em 30s). Aumento abrupto de CPU detectado.",
		Actions: []string{
			"Verificar se houve aumento de tráfego súbito",
			"Verificar logs da aplicação para erros ou slow queries",
			"Monitorar se HPA vai escalar adequadamente",
		},
		Snapshot: &models.HPASnapshot{
			CurrentReplicas: 8,
			MinReplicas:     2,
			MaxReplicas:     10,
			CPUCurrent:      95.0,
			MemoryCurrent:   70.0,
			ErrorRate:       2.5,
			P95Latency:      450.0,
		},
	}

	select {
	case m.GetAnomalyChan() <- anomaly2:
		log.Info().Msg("Anomalia CPU_SPIKE enviada")
	default:
		log.Warn().Msg("Canal de anomalias cheio")
	}

	time.Sleep(1 * time.Second)

	// Anomalia 3: HIGH_ERROR_RATE
	anomaly3 := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeHighErrorRate,
		Severity:  models.SeverityCritical,
		Cluster:   clusters[2],
		Namespace: namespaces[2],
		HPAName:   hpaNames[2],
		Timestamp: time.Now(),
		Message:   "Taxa de erros alta: 8.50% (limite: 5.00%). Requer atenção imediata!",
		Actions: []string{
			"Verificar logs de erros 5xx imediatamente",
			"Verificar health checks dos pods",
			"Verificar conectividade com dependências (DB, APIs)",
			"Considerar rollback se deploy recente",
		},
		Snapshot: &models.HPASnapshot{
			CurrentReplicas: 10,
			MinReplicas:     2,
			MaxReplicas:     10,
			CPUCurrent:      85.0,
			MemoryCurrent:   80.0,
			ErrorRate:       8.5,
			P95Latency:      1200.0,
		},
	}

	select {
	case m.GetAnomalyChan() <- anomaly3:
		log.Info().Msg("Anomalia HIGH_ERROR_RATE enviada")
	default:
		log.Warn().Msg("Canal de anomalias cheio")
	}

	time.Sleep(1 * time.Second)

	// Anomalia 4: MAXED_OUT
	anomaly4 := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeMaxedOut,
		Severity:  models.SeverityCritical,
		Cluster:   clusters[0],
		Namespace: namespaces[2],
		HPAName:   hpaNames[3],
		Timestamp: time.Now(),
		Message:   "HPA no limite máximo (10/10 réplicas) com CPU 92.0% (target: 70%). Capacidade esgotada!",
		Actions: []string{
			"URGENTE: Aumentar maxReplicas do HPA",
			"Verificar se é spike temporário ou crescimento real",
			"Considerar otimização da aplicação",
			"Avaliar vertical scaling (aumentar resources)",
		},
		Snapshot: &models.HPASnapshot{
			CurrentReplicas: 10,
			MinReplicas:     2,
			MaxReplicas:     10,
			CPUCurrent:      92.0,
			MemoryCurrent:   75.0,
			ErrorRate:       1.2,
			P95Latency:      650.0,
		},
	}

	select {
	case m.GetAnomalyChan() <- anomaly4:
		log.Info().Msg("Anomalia MAXED_OUT enviada")
	default:
		log.Warn().Msg("Canal de anomalias cheio")
	}

	log.Info().Msg("Geração de dados de teste concluída")

	// Continua gerando snapshots periodicamente
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Atualiza snapshots existentes
		for _, cluster := range clusters {
			snapshot := &models.HPASnapshot{
				Timestamp:       time.Now(),
				Cluster:         cluster,
				Namespace:       namespaces[0],
				Name:            hpaNames[0],
				MinReplicas:     2,
				MaxReplicas:     10,
				CurrentReplicas: 3,
				CPUCurrent:      60.0 + float64(time.Now().Unix()%20),
				MemoryCurrent:   65.0,
				DataSource:      models.DataSourcePrometheus,
			}

			select {
			case m.GetSnapshotChan() <- snapshot:
			default:
			}
		}
	}
}
