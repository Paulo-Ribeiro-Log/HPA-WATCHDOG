package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/portforward"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/scanner"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ScanEngine orquestra coleta, análise e detecção
type ScanEngine struct {
	config *scanner.ScanConfig

	// Componentes
	pfManager   *portforward.PortForwardManager
	cache       *storage.TimeSeriesCache
	persistence *storage.Persistence
	detector    *analyzer.Detector

	// Stress Test (apenas em ScanModeStressTest)
	baselineCollector *monitor.BaselineCollector
	stressComparator  *analyzer.StressComparator
	baseline          *models.BaselineSnapshot
	stressMetrics     *models.StressTestMetrics
	testID            string
	testStartTime     time.Time

	// Canais de saída
	snapshotChan chan *models.HPASnapshot
	anomalyChan  chan analyzer.Anomaly

	// Controle
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	paused   bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// New cria novo scan engine
func New(cfg *scanner.ScanConfig, snapshotChan chan *models.HPASnapshot, anomalyChan chan analyzer.Anomaly) *ScanEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// Cria cache
	cache := storage.NewTimeSeriesCache(nil)

	// Cria e configura persistência SQLite
	persistence, err := storage.NewPersistence(storage.DefaultPersistenceConfig())
	if err != nil {
		log.Warn().Err(err).Msg("Falha ao criar persistência, continuando sem SQLite")
		persistence = nil
	} else {
		// Configura persistência no cache (habilita auto-save/auto-load)
		cache.SetPersistence(persistence)
		log.Info().Msg("Persistência SQLite habilitada com auto-save")
	}

	detector := analyzer.NewDetector(cache, nil)

	return &ScanEngine{
		config:       cfg,
		pfManager:    portforward.NewManager(),
		cache:        cache,
		persistence:  persistence,
		detector:     detector,
		snapshotChan: snapshotChan,
		anomalyChan:  anomalyChan,
		ctx:          ctx,
		cancel:       cancel,
		stopChan:     make(chan struct{}),
	}
}

// Start inicia scan engine
func (e *ScanEngine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.paused = false
	e.mu.Unlock()

	log.Info().
		Str("mode", e.config.Mode.String()).
		Dur("interval", e.config.Interval).
		Dur("duration", e.config.Duration).
		Msg("Iniciando scan engine")

	// Inicia port-forwards para todos os clusters
	for _, target := range e.config.Targets {
		if err := e.pfManager.Start(target.Cluster); err != nil {
			log.Error().
				Err(err).
				Str("cluster", target.Cluster).
				Msg("Falha ao iniciar port-forward")
			// Continua com outros clusters
		}
	}

	// Se modo stress test, captura baseline antes de iniciar
	if e.config.Mode == scanner.ScanModeStressTest {
		if err := e.captureBaseline(); err != nil {
			log.Error().
				Err(err).
				Msg("Falha ao capturar baseline, continuando sem baseline")
		}
	}

	// Inicia loop de scan
	e.wg.Add(1)
	go e.scanLoop()

	return nil
}

// Stop para scan engine
func (e *ScanEngine) Stop() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()

	log.Info().Msg("Parando scan engine")

	// Cancela contexto
	e.cancel()

	// Para port-forwards
	if err := e.pfManager.StopAll(); err != nil {
		log.Error().Err(err).Msg("Erro ao parar port-forwards")
	}

	// Aguarda goroutines
	e.wg.Wait()

	// Se modo stress test, finaliza e salva resultado
	if e.config.Mode == scanner.ScanModeStressTest {
		if err := e.finalizeStressTest(); err != nil {
			log.Error().
				Err(err).
				Msg("Erro ao finalizar stress test")
		}
	}

	// Cleanup e fecha persistência
	if e.persistence != nil {
		if err := e.persistence.Cleanup(); err != nil {
			log.Warn().Err(err).Msg("Erro ao limpar dados antigos")
		}
		if err := e.persistence.Close(); err != nil {
			log.Warn().Err(err).Msg("Erro ao fechar banco de dados")
		}
		log.Info().Msg("Persistência SQLite fechada")
	}

	log.Info().Msg("Scan engine parado")
	return nil
}

// Pause pausa scans (mantém port-forwards ativos)
func (e *ScanEngine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running && !e.paused {
		e.paused = true
		log.Info().Msg("Scan pausado")
	}
}

// Resume retoma scans
func (e *ScanEngine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running && e.paused {
		e.paused = false
		log.Info().Msg("Scan retomado")
	}
}

// IsRunning retorna se engine está rodando
func (e *ScanEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// IsPaused retorna se engine está pausado
func (e *ScanEngine) IsPaused() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.paused
}

// scanLoop loop principal de scan
func (e *ScanEngine) scanLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.Interval)
	defer ticker.Stop()

	// Primeiro scan imediato
	e.runScan()

	// Controle de duração
	var durationTimer *time.Timer
	if e.config.Duration > 0 {
		durationTimer = time.NewTimer(e.config.Duration)
		defer durationTimer.Stop()
	}

	scanCount := 0
	maxScans := e.config.EstimateScans()

	for {
		select {
		case <-e.ctx.Done():
			log.Info().Msg("Scan loop encerrado (context cancelled)")
			return

		case <-durationTimer.C:
			if durationTimer != nil {
				log.Info().Msg("Duração máxima atingida, parando scans")
				e.Stop()
				return
			}

		case <-ticker.C:
			// Verifica se pausado
			e.mu.RLock()
			paused := e.paused
			e.mu.RUnlock()

			if paused {
				log.Debug().Msg("Scan pausado, aguardando...")
				continue
			}

			// Verifica limite de scans
			scanCount++
			if maxScans > 0 && scanCount >= maxScans {
				log.Info().
					Int("scans", scanCount).
					Msg("Número máximo de scans atingido")
				e.Stop()
				return
			}

			e.runScan()
		}
	}
}

// runScan executa um scan completo
func (e *ScanEngine) runScan() {
	log.Info().Msg("Executando scan...")

	scanStart := time.Now()

	// Para cada target configurado
	for _, target := range e.config.Targets {
		log.Info().
			Str("cluster", target.Cluster).
			Strs("namespaces", target.Namespaces).
			Msg("Escaneando cluster")

		// Cria contexto com timeout para o scan
		ctx, cancel := context.WithTimeout(e.ctx, 2*time.Minute)

		// Obtém URL do Prometheus (port-forward)
		promEndpoint := e.pfManager.GetURL(target.Cluster)
		if promEndpoint == "" {
			log.Warn().
				Str("cluster", target.Cluster).
				Msg("Port-forward não disponível, pulando cluster")
			cancel()
			continue
		}

		// Cria ClusterInfo
		clusterInfo := &models.ClusterInfo{
			Name:    target.Cluster,
			Context: target.Cluster, // Assumindo que cluster name = context name
		}

		// Cria collector para este cluster
		collector, err := monitor.NewCollector(clusterInfo, promEndpoint, &monitor.CollectorConfig{
			ScanInterval:      e.config.Interval,
			ExcludeNamespaces: []string{},
			EnablePrometheus:  true,
		})
		if err != nil {
			log.Error().
				Err(err).
				Str("cluster", target.Cluster).
				Msg("Falha ao criar collector")
			cancel()
			continue
		}

		// Executa scan do cluster
		result, err := collector.Scan(ctx)
		if err != nil {
			log.Error().
				Err(err).
				Str("cluster", target.Cluster).
				Msg("Falha ao executar scan")
			cancel()
			continue
		}

		// Envia snapshots coletados para canal da TUI
		snapshots := collector.GetCache().GetAll()
		snapshotList := make([]*models.HPASnapshot, 0, len(snapshots))

		for _, ts := range snapshots {
			latest := ts.GetLatest()
			if latest != nil {
				snapshotList = append(snapshotList, latest)

				// Envia snapshot para canal (non-blocking)
				select {
				case e.snapshotChan <- latest:
				default:
					log.Warn().
						Str("cluster", target.Cluster).
						Msg("Canal de snapshots cheio, descartando snapshot")
				}
			}
		}

		// Se modo stress test, compara com baseline
		if e.config.Mode == scanner.ScanModeStressTest && e.stressComparator != nil {
			e.compareWithBaseline(snapshotList)
		}

		// Envia anomalias detectadas para canal da TUI
		for _, anomaly := range result.Anomalies {
			select {
			case e.anomalyChan <- anomaly:
			default:
				log.Warn().
					Str("cluster", target.Cluster).
					Msg("Canal de anomalias cheio, descartando anomalia")
			}
		}

		log.Info().
			Str("cluster", target.Cluster).
			Int("snapshots", result.SnapshotsCount).
			Int("anomalies", len(result.Anomalies)).
			Int("errors", len(result.Errors)).
			Msg("Cluster escaneado com sucesso")

		cancel()
	}

	scanDuration := time.Since(scanStart)
	log.Info().
		Dur("duration", scanDuration).
		Msg("Scan completo")
}

// captureBaseline captura baseline antes do stress test
func (e *ScanEngine) captureBaseline() error {
	log.Info().Msg("Capturando baseline antes do stress test...")

	// Gera ID único para o teste
	e.testID = uuid.New().String()
	e.testStartTime = time.Now()

	// Inicializar StressTestMetrics
	e.stressMetrics = models.NewStressTestMetrics(
		fmt.Sprintf("Stress Test %s", e.testStartTime.Format("2006-01-02 15:04:05")),
		e.testStartTime,
		e.config.Interval,
	)

	// Para cada target, captura baseline
	for _, target := range e.config.Targets {
		promEndpoint := e.pfManager.GetURL(target.Cluster)
		if promEndpoint == "" {
			log.Warn().
				Str("cluster", target.Cluster).
				Msg("Port-forward não disponível, pulando baseline deste cluster")
			continue
		}

		// Cria ClusterInfo
		clusterInfo := &models.ClusterInfo{
			Name:    target.Cluster,
			Context: target.Cluster,
		}

		// Cria collector temporário para baseline
		collector, err := monitor.NewCollector(clusterInfo, promEndpoint, &monitor.CollectorConfig{
			ScanInterval:      e.config.Interval,
			ExcludeNamespaces: []string{},
			EnablePrometheus:  true,
		})
		if err != nil {
			log.Warn().
				Err(err).
				Str("cluster", target.Cluster).
				Msg("Falha ao criar collector para baseline")
			continue
		}

		// Cria baseline collector
		e.baselineCollector = monitor.NewBaselineCollector(
			collector.GetPrometheusClient(),
			collector.GetK8sClient(),
		)

		// Captura baseline (últimos 30min)
		ctx, cancel := context.WithTimeout(e.ctx, 5*time.Minute)
		baseline, err := e.baselineCollector.CaptureBaseline(ctx, 30*time.Minute)
		cancel()

		if err != nil {
			log.Error().
				Err(err).
				Str("cluster", target.Cluster).
				Msg("Falha ao capturar baseline")
			continue
		}

		// Salva baseline (primeiro cluster ou merge se múltiplos)
		if e.baseline == nil {
			e.baseline = baseline
		} else {
			// Merge baselines de múltiplos clusters
			e.baseline.TotalHPAs += baseline.TotalHPAs
			e.baseline.TotalReplicas += baseline.TotalReplicas
			for key, hpaBaseline := range baseline.HPABaselines {
				e.baseline.HPABaselines[key] = hpaBaseline
			}
		}

		log.Info().
			Str("cluster", target.Cluster).
			Int("hpas", baseline.TotalHPAs).
			Int("replicas", baseline.TotalReplicas).
			Msg("Baseline capturado com sucesso")
	}

	if e.baseline == nil {
		return fmt.Errorf("falha ao capturar baseline de qualquer cluster")
	}

	// Cria comparador com baseline
	e.stressComparator = analyzer.NewStressComparator(e.baseline, nil)

	// Salva baseline no SQLite
	if e.persistence != nil {
		if err := e.persistence.SaveBaseline(e.testID, e.baseline); err != nil {
			log.Warn().
				Err(err).
				Msg("Falha ao salvar baseline no SQLite")
		} else {
			log.Info().
				Str("test_id", e.testID).
				Msg("Baseline salvo no SQLite")
		}
	}

	// Inicializar métricas PRE
	e.stressMetrics.PeakMetrics.TotalReplicasPre = e.baseline.TotalReplicas
	e.stressMetrics.TotalClusters = len(e.config.Targets)
	e.stressMetrics.TotalHPAs = e.baseline.TotalHPAs

	log.Info().
		Str("test_id", e.testID).
		Int("total_hpas", e.baseline.TotalHPAs).
		Int("total_replicas", e.baseline.TotalReplicas).
		Msg("Baseline capturado e stress test iniciado")

	return nil
}

// compareWithBaseline compara snapshots atuais com baseline
func (e *ScanEngine) compareWithBaseline(snapshots []*models.HPASnapshot) {
	if e.stressComparator == nil || len(snapshots) == 0 {
		return
	}

	// Compara todos os snapshots com baseline
	results := e.stressComparator.CompareMultiple(snapshots)

	// Gera resumo
	summary := e.stressComparator.GetSummary(results)

	// Atualiza métricas de pico
	e.mu.Lock()
	defer e.mu.Unlock()

	// Calcula réplicas atuais totais
	var totalReplicasCurrent int32
	for _, snapshot := range snapshots {
		totalReplicasCurrent += snapshot.CurrentReplicas
	}

	// Calcula réplicas atuais totais em int
	totalReplicasCurrentInt := int(totalReplicasCurrent)

	// Atualiza pico de réplicas
	if totalReplicasCurrentInt > e.stressMetrics.PeakMetrics.TotalReplicasPeak {
		e.stressMetrics.PeakMetrics.TotalReplicasPeak = totalReplicasCurrentInt
	}

	// Atualiza scans e métricas totais
	e.stressMetrics.TotalScans++
	e.stressMetrics.TotalHPAsMonitored = len(snapshots)

	// Atualiza pico de CPU e Memory
	for _, result := range results {
		hpaKey := fmt.Sprintf("%s/%s/%s", result.Cluster, result.Namespace, result.HPA)

		if result.CPUCurrent > e.stressMetrics.PeakMetrics.MaxCPUPercent {
			e.stressMetrics.PeakMetrics.MaxCPUPercent = result.CPUCurrent
			e.stressMetrics.PeakMetrics.MaxCPUHPA = hpaKey
			e.stressMetrics.PeakMetrics.MaxCPUTime = result.Timestamp
		}

		if result.MemoryCurrent > e.stressMetrics.PeakMetrics.MaxMemoryPercent {
			e.stressMetrics.PeakMetrics.MaxMemoryPercent = result.MemoryCurrent
			e.stressMetrics.PeakMetrics.MaxMemoryHPA = hpaKey
			e.stressMetrics.PeakMetrics.MaxMemoryTime = result.Timestamp
		}
	}

	// Atualiza métricas de saúde
	e.stressMetrics.TotalClusters = len(e.config.Targets)
	e.stressMetrics.TotalHPAs = summary.TotalHPAs
	e.stressMetrics.TotalHPAsWithIssues = summary.DegradedCount + summary.CriticalCount

	// Salva snapshots no SQLite
	if e.persistence != nil {
		for _, snapshot := range snapshots {
			if err := e.persistence.SaveStressTestSnapshot(e.testID, snapshot); err != nil {
				log.Debug().
					Err(err).
					Str("hpa", snapshot.Name).
					Msg("Falha ao salvar snapshot do stress test")
			}
		}
	}

	// Log de progresso
	log.Info().
		Str("test_id", e.testID).
		Int("total_hpas", summary.TotalHPAs).
		Int("normal", summary.NormalCount).
		Int("degraded", summary.DegradedCount).
		Int("critical", summary.CriticalCount).
		Float64("health", summary.HealthPercentage).
		Msg("Comparação com baseline executada")
}

// finalizeStressTest finaliza stress test e salva resultado
func (e *ScanEngine) finalizeStressTest() error {
	if e.stressMetrics == nil {
		log.Warn().Msg("Nenhuma métrica de stress test para finalizar")
		return nil
	}

	log.Info().
		Str("test_id", e.testID).
		Msg("Finalizando stress test...")

	e.mu.Lock()
	defer e.mu.Unlock()

	// Finaliza métricas usando método Complete()
	e.stressMetrics.Complete()

	// Calcula réplicas POST
	snapshots := e.cache.GetAll()
	var totalReplicasPost int32
	for _, ts := range snapshots {
		latest := ts.GetLatest()
		if latest != nil {
			totalReplicasPost += latest.CurrentReplicas
		}
	}
	e.stressMetrics.PeakMetrics.TotalReplicasPost = int(totalReplicasPost)

	// Calcula aumento de réplicas
	e.stressMetrics.PeakMetrics.ReplicaIncrease = e.stressMetrics.PeakMetrics.TotalReplicasPeak - e.stressMetrics.PeakMetrics.TotalReplicasPre
	if e.stressMetrics.PeakMetrics.TotalReplicasPre > 0 {
		e.stressMetrics.PeakMetrics.ReplicaIncreaseP = (float64(e.stressMetrics.PeakMetrics.ReplicaIncrease) / float64(e.stressMetrics.PeakMetrics.TotalReplicasPre)) * 100
	}

	// Salva resultado no SQLite
	if e.persistence != nil {
		if err := e.persistence.SaveStressTestResult(e.testID, e.stressMetrics); err != nil {
			log.Error().
				Err(err).
				Msg("Falha ao salvar resultado do stress test")
			return err
		}

		log.Info().
			Str("test_id", e.testID).
			Msg("Resultado do stress test salvo no SQLite")
	}

	// Log do resumo final
	log.Info().
		Str("test_id", e.testID).
		Str("test_name", e.stressMetrics.TestName).
		Dur("duration", e.stressMetrics.Duration).
		Str("status", string(e.stressMetrics.Status)).
		Int("total_hpas", e.stressMetrics.TotalHPAs).
		Int("hpas_with_issues", e.stressMetrics.TotalHPAsWithIssues).
		Int("replicas_pre", e.stressMetrics.PeakMetrics.TotalReplicasPre).
		Int("replicas_peak", e.stressMetrics.PeakMetrics.TotalReplicasPeak).
		Int("replicas_post", e.stressMetrics.PeakMetrics.TotalReplicasPost).
		Int("replica_increase", e.stressMetrics.PeakMetrics.ReplicaIncrease).
		Float64("replica_increase_percent", e.stressMetrics.PeakMetrics.ReplicaIncreaseP).
		Msg("Stress test finalizado")

	return nil
}
