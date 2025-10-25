package engine

import (
	"context"
	"sync"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/portforward"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/scanner"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/storage"
	"github.com/rs/zerolog/log"
)

// ScanEngine orquestra coleta, análise e detecção
type ScanEngine struct {
	config *scanner.ScanConfig

	// Componentes
	pfManager *portforward.PortForwardManager
	cache     *storage.TimeSeriesCache
	detector  *analyzer.Detector

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

	cache := storage.NewTimeSeriesCache(nil)
	detector := analyzer.NewDetector(cache, nil)

	return &ScanEngine{
		config:       cfg,
		pfManager:    portforward.NewManager(),
		cache:        cache,
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

	// TODO: Implementar coleta real via Prometheus
	// Por enquanto, só loga
	for _, target := range e.config.Targets {
		log.Info().
			Str("cluster", target.Cluster).
			Strs("namespaces", target.Namespaces).
			Msg("Escaneando cluster")

		// TODO: Chamar collector para coletar snapshots
		// TODO: Enviar snapshots para cache
		// TODO: Rodar detector
		// TODO: Enviar anomalias encontradas
	}

	scanDuration := time.Since(scanStart)
	log.Info().
		Dur("duration", scanDuration).
		Msg("Scan completo")
}
