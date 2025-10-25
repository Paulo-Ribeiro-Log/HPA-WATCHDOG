package analyzer

import (
	"fmt"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/storage"
	"github.com/rs/zerolog/log"
)

// Detector detecta anomalias em HPAs
type Detector struct {
	cache  *storage.TimeSeriesCache
	config *DetectorConfig
}

// DetectorConfig configuração do detector
type DetectorConfig struct {
	// Oscillation
	OscillationMaxChanges int           // Máximo de mudanças de réplicas permitidas
	OscillationWindow     time.Duration // Janela de tempo para análise

	// Maxed Out
	MaxedOutCPUDeviation float64       // % acima do target para considerar maxed out
	MaxedOutMinDuration  time.Duration // Tempo mínimo no estado maxed out

	// High Error Rate
	ErrorRateThreshold  float64       // % de erros para alertar
	ErrorRateMinDuration time.Duration // Tempo mínimo com erros altos

	// Pods Not Ready
	NotReadyThreshold    float64       // % mínimo de pods ready
	NotReadyMinDuration  time.Duration // Tempo mínimo com pods not ready

	// Cooldown para evitar alertas duplicados
	AlertCooldown time.Duration // Tempo entre alertas do mesmo tipo
}

// DefaultDetectorConfig retorna configuração padrão
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		// Oscillation: >5 mudanças em 5min
		OscillationMaxChanges: 5,
		OscillationWindow:     5 * time.Minute,

		// Maxed Out: CPU >target+20% por 2min
		MaxedOutCPUDeviation: 20.0,
		MaxedOutMinDuration:  2 * time.Minute,

		// High Error Rate: >5% por 2min
		ErrorRateThreshold:   5.0,
		ErrorRateMinDuration: 2 * time.Minute,

		// Pods Not Ready: <70% por 3min
		NotReadyThreshold:    70.0,
		NotReadyMinDuration:  3 * time.Minute,

		// Cooldown: 5min entre alertas do mesmo tipo
		AlertCooldown: 5 * time.Minute,
	}
}

// NewDetector cria novo detector
func NewDetector(cache *storage.TimeSeriesCache, config *DetectorConfig) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}

	log.Info().
		Int("oscillation_max_changes", config.OscillationMaxChanges).
		Float64("maxed_out_cpu_deviation", config.MaxedOutCPUDeviation).
		Float64("error_rate_threshold", config.ErrorRateThreshold).
		Float64("not_ready_threshold", config.NotReadyThreshold).
		Msg("Anomaly Detector initialized")

	return &Detector{
		cache:  cache,
		config: config,
	}
}

// DetectionResult resultado da detecção
type DetectionResult struct {
	Anomalies []Anomaly
	Checked   int
	Timestamp time.Time
}

// Anomaly representa uma anomalia detectada
type Anomaly struct {
	Type        AnomalyType
	Severity    models.AlertSeverity
	Cluster     string
	Namespace   string
	HPAName     string
	Timestamp   time.Time
	Message     string
	Description string
	Snapshot    *models.HPASnapshot
	Stats       *models.HPAStats
	Actions     []string
}

// AnomalyType tipos de anomalias
type AnomalyType string

const (
	AnomalyTypeOscillation   AnomalyType = "OSCILLATION"
	AnomalyTypeMaxedOut      AnomalyType = "MAXED_OUT"
	AnomalyTypeOOMKilled     AnomalyType = "OOM_KILLED"
	AnomalyTypePodsNotReady  AnomalyType = "PODS_NOT_READY"
	AnomalyTypeHighErrorRate AnomalyType = "HIGH_ERROR_RATE"
)

// Detect executa detecção em todos os HPAs
func (d *Detector) Detect() *DetectionResult {
	result := &DetectionResult{
		Anomalies: []Anomaly{},
		Timestamp: time.Now(),
	}

	allData := d.cache.GetAll()
	result.Checked = len(allData)

	for key, ts := range allData {
		// Precisa de pelo menos 1 snapshot
		if len(ts.Snapshots) == 0 {
			continue
		}

		latest := ts.GetLatest()
		if latest == nil {
			continue
		}

		// Fase 1 - MVP: 5 anomalias críticas
		anomalies := []Anomaly{}

		// 1. Oscillation
		if anomaly := d.detectOscillation(ts, latest); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}

		// 2. Maxed Out
		if anomaly := d.detectMaxedOut(ts, latest); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}

		// 3. OOMKilled (detectado via K8s events - implementar depois)
		// TODO: Integrar com K8s events

		// 4. Pods Not Ready
		if anomaly := d.detectPodsNotReady(ts, latest); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}

		// 5. High Error Rate
		if anomaly := d.detectHighErrorRate(ts, latest); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}

		result.Anomalies = append(result.Anomalies, anomalies...)

		if len(anomalies) > 0 {
			log.Debug().
				Str("hpa", key).
				Int("anomalies", len(anomalies)).
				Msg("Anomalies detected")
		}
	}

	if len(result.Anomalies) > 0 {
		log.Info().
			Int("total", len(result.Anomalies)).
			Int("checked", result.Checked).
			Msg("Anomaly detection complete")
	}

	return result
}

// detectOscillation detecta oscillation (>5 mudanças de réplicas em 5min)
func (d *Detector) detectOscillation(ts *models.TimeSeriesData, latest *models.HPASnapshot) *Anomaly {
	// Usa as stats calculadas pelo cache
	if ts.Stats.ReplicaChanges <= d.config.OscillationMaxChanges {
		return nil
	}

	return &Anomaly{
		Type:      AnomalyTypeOscillation,
		Severity:  models.SeverityCritical,
		Cluster:   latest.Cluster,
		Namespace: latest.Namespace,
		HPAName:   latest.Name,
		Timestamp: time.Now(),
		Message: fmt.Sprintf("HPA oscilando: %d mudanças de réplicas em %v",
			ts.Stats.ReplicaChanges, d.config.OscillationWindow),
		Description: fmt.Sprintf(
			"O HPA mudou de réplicas %d vezes nos últimos %v (limite: %d). "+
				"Isso indica configuração instável ou carga muito variável.",
			ts.Stats.ReplicaChanges,
			d.config.OscillationWindow,
			d.config.OscillationMaxChanges,
		),
		Snapshot: latest,
		Stats:    &ts.Stats,
		Actions: []string{
			"Aumentar HPA stabilizationWindow (scaleDown: 300s)",
			"Revisar targets de CPU/Memory (podem estar muito sensíveis)",
			"Verificar se carga é realmente variável ou há problema na aplicação",
			"Considerar usar behavior policies (v2beta2+)",
		},
	}
}

// detectMaxedOut detecta maxed out (réplicas=max + CPU>target+20%)
func (d *Detector) detectMaxedOut(ts *models.TimeSeriesData, latest *models.HPASnapshot) *Anomaly {
	// Verifica se está no máximo de réplicas
	if latest.CurrentReplicas < latest.MaxReplicas {
		return nil
	}

	// Verifica se CPU está acima do target + deviation
	if latest.CPUTarget == 0 || latest.CPUCurrent == 0 {
		return nil
	}

	cpuDeviation := latest.CPUCurrent - float64(latest.CPUTarget)
	if cpuDeviation < d.config.MaxedOutCPUDeviation {
		return nil
	}

	// Verifica duração mínima (precisa estar maxed out há pelo menos 2min)
	if !d.checkMinDuration(ts, d.config.MaxedOutMinDuration, func(s *models.HPASnapshot) bool {
		return s.CurrentReplicas >= s.MaxReplicas &&
		       s.CPUCurrent > float64(s.CPUTarget)+d.config.MaxedOutCPUDeviation
	}) {
		return nil
	}

	return &Anomaly{
		Type:      AnomalyTypeMaxedOut,
		Severity:  models.SeverityCritical,
		Cluster:   latest.Cluster,
		Namespace: latest.Namespace,
		HPAName:   latest.Name,
		Timestamp: time.Now(),
		Message: fmt.Sprintf("HPA no limite máximo: %d réplicas, CPU %.2f%% (target: %d%%)",
			latest.MaxReplicas, latest.CPUCurrent, latest.CPUTarget),
		Description: fmt.Sprintf(
			"HPA atingiu o limite máximo de %d réplicas e CPU está em %.2f%% "+
				"(%.2f%% acima do target de %d%%). Não consegue escalar mais.",
			latest.MaxReplicas,
			latest.CPUCurrent,
			cpuDeviation,
			latest.CPUTarget,
		),
		Snapshot: latest,
		Stats:    &ts.Stats,
		Actions: []string{
			fmt.Sprintf("URGENTE: Aumentar maxReplicas de %d para %d ou mais",
				latest.MaxReplicas, latest.MaxReplicas*2),
			"Verificar se cluster tem capacidade suficiente",
			"Considerar escalar verticalmente (aumentar resources por pod)",
			"Investigar se há gargalo além de CPU (DB, API externa, etc)",
		},
	}
}

// detectPodsNotReady detecta pods not ready (<70% ready por 3min)
func (d *Detector) detectPodsNotReady(ts *models.TimeSeriesData, latest *models.HPASnapshot) *Anomaly {
	if latest.CurrentReplicas == 0 {
		return nil
	}

	// Por enquanto, usa apenas o campo Ready
	// TODO: Implementar contagem real de pods ready via K8s API
	if latest.Ready {
		return nil
	}

	// Verifica duração mínima
	if !d.checkMinDuration(ts, d.config.NotReadyMinDuration, func(s *models.HPASnapshot) bool {
		return !s.Ready
	}) {
		return nil
	}

	return &Anomaly{
		Type:      AnomalyTypePodsNotReady,
		Severity:  models.SeverityCritical,
		Cluster:   latest.Cluster,
		Namespace: latest.Namespace,
		HPAName:   latest.Name,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Pods não estão prontos há mais de %v", d.config.NotReadyMinDuration),
		Description: fmt.Sprintf(
			"HPA tem %d réplicas mas pods não estão passando readiness probe. "+
				"Problema persiste há mais de %v.",
			latest.CurrentReplicas,
			d.config.NotReadyMinDuration,
		),
		Snapshot: latest,
		Stats:    &ts.Stats,
		Actions: []string{
			"Verificar logs dos pods: kubectl logs -n " + latest.Namespace,
			"Verificar readiness probe: kubectl describe pod",
			"Verificar dependências externas (DB, cache, APIs)",
			"Ajustar readiness probe se muito sensível",
		},
	}
}

// detectHighErrorRate detecta high error rate (>5% por 2min)
func (d *Detector) detectHighErrorRate(ts *models.TimeSeriesData, latest *models.HPASnapshot) *Anomaly {
	// Precisa ter métricas do Prometheus
	if latest.DataSource != models.DataSourcePrometheus {
		return nil
	}

	if latest.ErrorRate < d.config.ErrorRateThreshold {
		return nil
	}

	// Verifica duração mínima
	if !d.checkMinDuration(ts, d.config.ErrorRateMinDuration, func(s *models.HPASnapshot) bool {
		return s.ErrorRate >= d.config.ErrorRateThreshold
	}) {
		return nil
	}

	return &Anomaly{
		Type:      AnomalyTypeHighErrorRate,
		Severity:  models.SeverityCritical,
		Cluster:   latest.Cluster,
		Namespace: latest.Namespace,
		HPAName:   latest.Name,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Taxa de erros alta: %.2f%% (limite: %.2f%%)",
			latest.ErrorRate, d.config.ErrorRateThreshold),
		Description: fmt.Sprintf(
			"Aplicação está retornando %.2f%% de erros 5xx há mais de %v. "+
				"Request rate: %.2f req/s, P95 latency: %.2fms",
			latest.ErrorRate,
			d.config.ErrorRateMinDuration,
			latest.RequestRate,
			latest.P95Latency,
		),
		Snapshot: latest,
		Stats:    &ts.Stats,
		Actions: []string{
			"Verificar logs de erro: kubectl logs -n " + latest.Namespace + " --tail=100 | grep ERROR",
			"Verificar dependências downstream (APIs, DB)",
			"Considerar scale up se erro relacionado a capacidade",
			"Verificar métricas de latência e throughput",
			"Analisar distributed tracing se disponível",
		},
	}
}

// checkMinDuration verifica se condição persiste por tempo mínimo
func (d *Detector) checkMinDuration(
	ts *models.TimeSeriesData,
	minDuration time.Duration,
	condition func(*models.HPASnapshot) bool,
) bool {
	if len(ts.Snapshots) < 2 {
		return false
	}

	// Usa o último snapshot como referência ao invés de time.Now()
	// Isso permite testes com snapshots no passado
	latest := ts.GetLatest()
	if latest == nil {
		return false
	}

	cutoff := latest.Timestamp.Add(-minDuration)

	// Conta quantos snapshots recentes satisfazem a condição
	satisfiedCount := 0
	var oldestSatisfied time.Time

	for i := len(ts.Snapshots) - 1; i >= 0; i-- {
		snapshot := &ts.Snapshots[i]

		// Ignora snapshots muito antigos
		if snapshot.Timestamp.Before(cutoff) {
			break
		}

		if condition(snapshot) {
			satisfiedCount++
			oldestSatisfied = snapshot.Timestamp
		} else {
			// Se encontrou snapshot que não satisfaz, para
			break
		}
	}

	// Verifica se todos os snapshots recentes satisfazem E
	// se o mais antigo está além do minDuration
	return satisfiedCount >= 2 && latest.Timestamp.Sub(oldestSatisfied) >= minDuration
}

// GetAnomalyCount retorna contagem de anomalias por tipo
func (result *DetectionResult) GetAnomalyCount() map[AnomalyType]int {
	counts := make(map[AnomalyType]int)

	for _, anomaly := range result.Anomalies {
		counts[anomaly.Type]++
	}

	return counts
}

// GetBySeverity filtra anomalias por severidade
func (result *DetectionResult) GetBySeverity(severity models.AlertSeverity) []Anomaly {
	filtered := []Anomaly{}

	for _, anomaly := range result.Anomalies {
		if anomaly.Severity == severity {
			filtered = append(filtered, anomaly)
		}
	}

	return filtered
}

// GetByCluster filtra anomalias por cluster
func (result *DetectionResult) GetByCluster(cluster string) []Anomaly {
	filtered := []Anomaly{}

	for _, anomaly := range result.Anomalies {
		if anomaly.Cluster == cluster {
			filtered = append(filtered, anomaly)
		}
	}

	return filtered
}
