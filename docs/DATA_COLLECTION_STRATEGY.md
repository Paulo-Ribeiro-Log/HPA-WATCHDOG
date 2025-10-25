# Estratégia de Coleta e Armazenamento de Dados

Arquitetura para coleta, armazenamento e análise de dados do HPA Watchdog.

## 🎯 Objetivos

1. **Performance** - Scan rápido (< 5s por cluster)
2. **Histórico** - Dados suficientes para detectar anomalias
3. **Baixo overhead** - Não sobrecarregar K8s API / Prometheus
4. **Análise temporal** - Comparar com baseline
5. **Persistence opcional** - In-memory primeiro, DB depois

---

## 📊 Modelo de Dados

### HPASnapshot (por scan)

```go
type HPASnapshot struct {
    // Metadata
    Timestamp   time.Time
    Cluster     string
    Namespace   string
    Name        string

    // K8s Data
    MinReplicas     int32
    MaxReplicas     int32
    CurrentReplicas int32
    DesiredReplicas int32
    CPUTarget       int32
    MemoryTarget    int32

    // Prometheus - Current
    CPUCurrent    float64
    MemoryCurrent float64

    // Prometheus - History (5min, 10 pontos)
    CPUHistory     []float64  // [t-300s, t-270s, ..., t-0s]
    MemoryHistory  []float64
    ReplicaHistory []int32

    // Extended Metrics
    RequestRate float64
    ErrorRate   float64
    P95Latency  float64

    // Resources
    CPURequest    string
    CPULimit      string
    MemoryRequest string
    MemoryLimit   string

    // Status
    Ready         bool
    ScalingActive bool
    LastScaleTime *time.Time

    DataSource DataSource // Prometheus, MetricsServer, Hybrid
}
```

### TimeSeriesData (in-memory cache)

```go
type TimeSeriesData struct {
    HPAKey      string // "cluster/namespace/name"
    Snapshots   []HPASnapshot
    MaxDuration time.Duration // 5 minutos

    // Estatísticas calculadas
    Stats HPAStats

    sync.RWMutex
}

type HPAStats struct {
    // Calculado dos últimos 5 min
    CPUAverage    float64
    CPUMin        float64
    CPUMax        float64
    CPUStdDev     float64

    ReplicaChanges int      // Quantas mudanças em 5min
    LastChange     time.Time

    // Trend
    CPUTrend      string // "increasing", "decreasing", "stable"
    ReplicaTrend  string
}
```

---

## 🔄 Fluxo de Coleta de Dados

### Opção 1: **In-Memory Only** (MVP - Recomendado) ⭐

```
┌─────────────────────────────────────────────────────────────┐
│                     SCAN LOOP (30s)                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Para cada cluster:                                      │
│     ├─ Lista HPAs                                           │
│     ├─ Para cada HPA:                                       │
│     │  ├─ Coleta K8s data                                   │
│     │  ├─ Coleta Prometheus metrics (se disponível)         │
│     │  ├─ Cria HPASnapshot                                  │
│     │  └─ Armazena in-memory                                │
│     │                                                        │
│  2. In-Memory Storage (5 minutos):                          │
│     ├─ TimeSeriesData map[string]*TimeSeriesData           │
│     ├─ Key: "cluster/namespace/hpa"                         │
│     ├─ Value: últimos 10 snapshots (5min @ 30s interval)   │
│     └─ Auto-cleanup: remove snapshots > 5min                │
│                                                              │
│  3. Análise (após cada scan):                               │
│     ├─ Calcula stats (avg, min, max, stddev)               │
│     ├─ Detecta anomalias comparando com histórico          │
│     ├─ Cria alerts se necessário                            │
│     └─ Envia para TUI                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   MEMORY STRUCTURE                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  timeSeriesCache = {                                        │
│      "prod/default/api": {                                  │
│          Snapshots: [                                       │
│              {t: 10:00:00, cpu: 70%, replicas: 5},         │
│              {t: 10:00:30, cpu: 72%, replicas: 5},         │
│              {t: 10:01:00, cpu: 75%, replicas: 6}, ← scale │
│              ...                                            │
│              {t: 10:05:00, cpu: 68%, replicas: 6},         │
│          ],                                                 │
│          Stats: {                                           │
│              CPUAverage: 71.2%,                            │
│              ReplicaChanges: 1,                            │
│              CPUTrend: "stable"                            │
│          }                                                  │
│      },                                                     │
│      "prod/default/worker": {...},                         │
│      ...                                                    │
│  }                                                          │
│                                                              │
│  Memory Usage:                                              │
│    - 250 HPAs × 10 snapshots × ~500 bytes = ~1.25 MB      │
│    - Totalmente aceitável!                                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Prós:**
- ✅ **Simples** - Sem DB, sem dependencies
- ✅ **Rápido** - Acesso instantâneo
- ✅ **Baixo overhead** - Só memória
- ✅ **Suficiente para MVP** - 5min histórico detecta maioria das anomalias

**Contras:**
- ❌ **Perde dados ao reiniciar** - Histórico perdido
- ❌ **Sem long-term trends** - Não detecta tendências de dias/semanas
- ❌ **Não persiste alertas** - Alertas somem ao reiniciar

**Quando usar:**
- ✅ **MVP / Fase 1**
- ✅ Detectar anomalias imediatas (oscillation, maxed out, etc)
- ✅ Monitoramento real-time

---

### Opção 2: **Hybrid (In-Memory + SQLite)** (Fase 2) 💾

```
┌─────────────────────────────────────────────────────────────┐
│                  HYBRID STORAGE                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  IN-MEMORY (hot data - 5 minutos):                         │
│  ├─ TimeSeriesData cache                                    │
│  ├─ Análise em tempo real                                   │
│  └─ Detecção de anomalias imediatas                        │
│      │                                                       │
│      │ (cada 5 minutos)                                     │
│      ▼                                                       │
│  SQLITE (warm data - 7 dias):                              │
│  ├─ snapshots table                                         │
│  ├─ anomalies table                                         │
│  ├─ baselines table (aprendizado)                          │
│  └─ alerts table (histórico)                               │
│      │                                                       │
│      │ (após 7 dias)                                        │
│      ▼                                                       │
│  ARCHIVE (cold data - opcional):                           │
│  └─ Compressed JSON/Parquet files                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Schema SQLite:**

```sql
-- Snapshots agregados (não todos os pontos)
CREATE TABLE snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    cluster TEXT NOT NULL,
    namespace TEXT NOT NULL,
    hpa_name TEXT NOT NULL,

    -- Aggregated metrics (5min window)
    cpu_avg REAL,
    cpu_min REAL,
    cpu_max REAL,
    cpu_p95 REAL,

    memory_avg REAL,
    memory_min REAL,
    memory_max REAL,

    replicas_current INTEGER,
    replicas_desired INTEGER,
    replica_changes INTEGER, -- mudanças nos últimos 5min

    request_rate REAL,
    error_rate REAL,
    p95_latency REAL,

    INDEX idx_hpa (cluster, namespace, hpa_name, timestamp)
);

-- Anomalias detectadas
CREATE TABLE anomalies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    detected_at DATETIME NOT NULL,
    resolved_at DATETIME,

    cluster TEXT NOT NULL,
    namespace TEXT NOT NULL,
    hpa_name TEXT NOT NULL,

    type TEXT NOT NULL, -- "oscillation", "maxed_out", etc
    severity TEXT NOT NULL, -- "critical", "warning", "info"

    description TEXT,
    suggested_action TEXT,

    -- Snapshot no momento da detecção
    snapshot_id INTEGER,

    -- Estado
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at DATETIME,
    acknowledged_by TEXT,

    INDEX idx_anomaly (cluster, namespace, hpa_name, detected_at)
);

-- Baselines (aprendizado de padrão normal)
CREATE TABLE baselines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster TEXT NOT NULL,
    namespace TEXT NOT NULL,
    hpa_name TEXT NOT NULL,

    -- Padrão temporal
    pattern_type TEXT, -- "hourly", "daily", "weekly"
    pattern_key TEXT,  -- "monday_09h", "weekday_peak", etc

    -- Estatísticas do padrão
    cpu_baseline REAL,
    cpu_stddev REAL,
    replicas_baseline INTEGER,

    samples INTEGER, -- quantas amostras formaram o baseline
    last_updated DATETIME,

    UNIQUE(cluster, namespace, hpa_name, pattern_type, pattern_key)
);

-- Histórico de alertas (para deduplicação)
CREATE TABLE alert_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,

    cluster TEXT NOT NULL,
    namespace TEXT NOT NULL,
    hpa_name TEXT NOT NULL,
    type TEXT NOT NULL,

    fingerprint TEXT NOT NULL, -- hash do alerta

    INDEX idx_fingerprint (fingerprint, created_at)
);
```

**Workflow:**

```go
// Scan loop
for {
    // 1. Coleta in-memory (como antes)
    snapshot := collectSnapshot(hpa)
    tsData.Add(snapshot)

    // 2. Detecta anomalias
    anomalies := analyzer.Detect(tsData)

    // 3. Persiste no SQLite (async, não bloqueia)
    go func() {
        // Salva snapshot agregado (cada 5min)
        if time.Since(lastPersist) > 5*time.Minute {
            db.SaveAggregatedSnapshot(tsData.Stats)
        }

        // Salva anomalias
        for _, anomaly := range anomalies {
            db.SaveAnomaly(anomaly)
        }
    }()

    // 4. Atualiza baseline (aprende padrão)
    if shouldUpdateBaseline() {
        baseline.Learn(tsData)
    }

    time.Sleep(30 * time.Second)
}
```

**Prós:**
- ✅ **Persistência** - Dados sobrevivem restart
- ✅ **Long-term trends** - Detecta padrões de dias/semanas
- ✅ **Baseline learning** - Aprende comportamento normal
- ✅ **Alert history** - Histórico completo
- ✅ **Leve** - SQLite é single-file, sem servidor

**Contras:**
- ⚠️ **Mais complexo** - Gerenciar DB, migrations
- ⚠️ **Disk I/O** - Pode ser gargalo (mitiga com async writes)
- ⚠️ **Single-node** - SQLite não é distribuído

**Quando usar:**
- ✅ **Fase 2** - Após MVP validado
- ✅ Baseline learning
- ✅ Histórico de alertas
- ✅ Trends de longo prazo

---

### Opção 3: **External TSDB (Prometheus)** (Avançado) 📈

```
┌─────────────────────────────────────────────────────────────┐
│              LEVERAGE PROMETHEUS                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Watchdog NÃO armazena métricas!                           │
│  ├─ Usa Prometheus como source of truth                    │
│  ├─ Queries PromQL para análise                            │
│  └─ Recording rules para agregações                         │
│                                                              │
│  Armazena APENAS:                                           │
│  ├─ Estado de alertas (in-memory)                          │
│  ├─ Baselines aprendidos (SQLite leve)                     │
│  └─ Configuração de thresholds                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Recording Rules no Prometheus:**

```yaml
# prometheus-rules.yaml
groups:
  - name: hpa_watchdog
    interval: 30s
    rules:
      # Replica changes in 5min
      - record: hpa_watchdog:replica_changes:5m
        expr: |
          changes(kube_horizontalpodautoscaler_status_current_replicas[5m])

      # CPU deviation from target
      - record: hpa_watchdog:cpu_deviation:percent
        expr: |
          (
            sum by (namespace, horizontalpodautoscaler) (
              rate(container_cpu_usage_seconds_total[1m])
            ) /
            sum by (namespace, horizontalpodautoscaler) (
              kube_pod_container_resource_requests{resource="cpu"}
            ) * 100
          ) -
          on(namespace, horizontalpodautoscaler)
          kube_horizontalpodautoscaler_spec_target_metric{metric_name="cpu"}

      # Maxed out detection
      - record: hpa_watchdog:maxed_out:bool
        expr: |
          (
            kube_horizontalpodautoscaler_status_current_replicas
            ==
            kube_horizontalpodautoscaler_spec_max_replicas
          )
          and
          (hpa_watchdog:cpu_deviation:percent > 20)
```

**Watchdog faz:**

```go
// Apenas queries, sem storage de métricas
func (d *Detector) DetectMaxedOut(cluster, namespace, hpa string) (*Anomaly, error) {
    query := `hpa_watchdog:maxed_out:bool{cluster="` + cluster + `",namespace="` + namespace + `",hpa="` + hpa + `"}`

    result, err := prometheus.Query(query)
    if err != nil {
        return nil, err
    }

    if result > 0 {
        // Cria anomaly
        return &Anomaly{
            Type: AnomalyMaxedOut,
            // ...
        }, nil
    }

    return nil, nil
}
```

**Prós:**
- ✅ **Zero storage** - Prometheus já tem tudo
- ✅ **Escalável** - Prometheus é TSDB otimizado
- ✅ **Rich queries** - PromQL poderoso
- ✅ **Já existe** - Aproveita infra existente

**Contras:**
- ⚠️ **Dependência** - Precisa de Prometheus
- ⚠️ **Latência** - Queries podem ser lentas
- ⚠️ **Complexidade** - Recording rules, federation

**Quando usar:**
- ✅ **Fase 3** - Produção em escala
- ✅ Múltiplos clusters
- ✅ Já tem Prometheus bem configurado

---

## 🎯 Detecção de Anomalias com Dados Coletados

### Exemplo: Oscillation Detection

**In-Memory (Opção 1):**

```go
func (d *Detector) DetectOscillation(tsData *TimeSeriesData) *Anomaly {
    // Últimos 10 snapshots (5 minutos)
    snapshots := tsData.GetHistory()

    // Conta mudanças de réplicas
    changes := 0
    for i := 1; i < len(snapshots); i++ {
        if snapshots[i].CurrentReplicas != snapshots[i-1].CurrentReplicas {
            changes++
        }
    }

    // Threshold: >5 mudanças em 5min
    if changes > 5 {
        return &Anomaly{
            Type:     AnomalyOscillation,
            Severity: SeverityCritical,
            Description: fmt.Sprintf(
                "HPA oscillating: %d replica changes in 5 minutes",
                changes,
            ),
            // Evidência
            Evidence: map[string]interface{}{
                "changes": changes,
                "history": extractReplicaHistory(snapshots),
            },
        }
    }

    return nil
}

// Helper
func extractReplicaHistory(snapshots []HPASnapshot) []int32 {
    history := make([]int32, len(snapshots))
    for i, s := range snapshots {
        history[i] = s.CurrentReplicas
    }
    return history
    // Ex: [3, 5, 3, 6, 4, 7, 3, 5, 4, 6] → 9 mudanças!
}
```

**Com SQLite (Opção 2) - Baseline Learning:**

```go
func (d *Detector) DetectMaxedOutWithBaseline(ctx context.Context, tsData *TimeSeriesData) *Anomaly {
    current := tsData.GetLatest()

    // Busca baseline do padrão atual
    baseline, err := d.db.GetBaseline(ctx,
        current.Cluster,
        current.Namespace,
        current.Name,
        getCurrentPattern(), // "monday_09h"
    )

    if err != nil || baseline == nil {
        // Sem baseline ainda, usa threshold fixo
        return d.detectMaxedOutSimple(current)
    }

    // Compara com baseline aprendido
    deviation := (current.CPUCurrent - baseline.CPUBaseline) / baseline.CPUStdDev

    // Maxed out + desvio significativo do normal
    if current.CurrentReplicas == current.MaxReplicas && deviation > 2.0 {
        return &Anomaly{
            Type:     AnomalyMaxedOut,
            Severity: SeverityCritical,
            Description: fmt.Sprintf(
                "HPA maxed out with CPU %.1f%% (%.1fσ above baseline %.1f%%)",
                current.CPUCurrent,
                deviation,
                baseline.CPUBaseline,
            ),
            Evidence: map[string]interface{}{
                "current_cpu": current.CPUCurrent,
                "baseline_cpu": baseline.CPUBaseline,
                "stddev": baseline.CPUStdDev,
                "deviation_sigma": deviation,
            },
        }
    }

    return nil
}

func getCurrentPattern() string {
    now := time.Now()

    // Padrões:
    // - "weekday_09h", "weekday_14h", etc (hora de pico)
    // - "weekend_low" (final de semana)
    // - "monday_peak" (segunda-feira)

    isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday
    if isWeekend {
        return "weekend_low"
    }

    hour := now.Hour()
    if hour >= 9 && hour <= 17 {
        return fmt.Sprintf("weekday_%02dh", hour)
    }

    return "weekday_off_hours"
}
```

---

## 📋 Recomendação de Implementação

### **Fase 1 - MVP** (IMPLEMENTAR AGORA)

**Storage:** In-Memory apenas

```go
// storage/memory.go
type MemoryStorage struct {
    timeSeries map[string]*TimeSeriesData
    mu         sync.RWMutex
}

func (s *MemoryStorage) Add(snapshot *HPASnapshot) {
    key := fmt.Sprintf("%s/%s/%s",
        snapshot.Cluster,
        snapshot.Namespace,
        snapshot.Name,
    )

    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.timeSeries[key]; !exists {
        s.timeSeries[key] = NewTimeSeriesData(key, 5*time.Minute)
    }

    s.timeSeries[key].Add(*snapshot)
}

func (s *MemoryStorage) Get(cluster, namespace, name string) *TimeSeriesData {
    key := fmt.Sprintf("%s/%s/%s", cluster, namespace, name)

    s.mu.RLock()
    defer s.mu.RUnlock()

    return s.timeSeries[key]
}
```

**Detector:**

```go
// analyzer/detector.go
type Detector struct {
    storage Storage
    config  *AnomalyConfig
}

func (d *Detector) Scan() []*Anomaly {
    var anomalies []*Anomaly

    // Para cada HPA monitorado
    for key, tsData := range d.storage.GetAll() {
        // Aplica cada regra
        if a := d.detectOscillation(tsData); a != nil {
            anomalies = append(anomalies, a)
        }

        if a := d.detectMaxedOut(tsData); a != nil {
            anomalies = append(anomalies, a)
        }

        // ... mais regras
    }

    return anomalies
}
```

**Vantagens:**
- ✅ Simples de implementar (1-2 dias)
- ✅ Sem dependências externas
- ✅ Performance excelente
- ✅ Suficiente para 90% dos casos

---

### **Fase 2 - Production** (Depois do MVP)

**Storage:** Hybrid (In-Memory + SQLite)

**Adicionar:**
- Persistence de snapshots agregados
- Baseline learning (7 dias de aprendizado)
- Alert history com deduplicação
- Recovery de estado após restart

**Timeline:** +3-5 dias após MVP

---

### **Fase 3 - Scale** (Futuro)

**Storage:** Leverage Prometheus + SQLite leve

**Adicionar:**
- Recording rules no Prometheus
- Queries otimizadas
- Multi-cluster federation
- Long-term storage (S3/GCS)

**Timeline:** 1-2 semanas

---

## 🧪 Comparação de Performance

| Métrica | In-Memory | SQLite | Prometheus |
|---------|-----------|--------|------------|
| **Write latency** | < 1ms | 5-10ms | 50-100ms |
| **Read latency** | < 1ms | 2-5ms | 100-500ms |
| **Memory (250 HPAs)** | ~1-2 MB | ~5-10 MB | N/A |
| **Disk usage** | 0 | ~100 MB/day | N/A |
| **Query power** | Simple | SQL | PromQL |
| **Restart recovery** | ❌ Lost | ✅ Full | ✅ Full |
| **Long-term trends** | ❌ 5min | ✅ Days | ✅ Weeks |

---

## 💡 Conclusão

### Para MVP (AGORA):
**🎯 Opção 1: In-Memory Only**

- Simples, rápido, sem dependencies
- 5 minutos de histórico = suficiente
- Detecta 90% das anomalias importantes
- Implementa em 1-2 dias

### Para Produção (DEPOIS):
**🎯 Opção 2: Hybrid (In-Memory + SQLite)**

- Persistence + long-term trends
- Baseline learning
- Alert history
- Implementa em +3-5 dias

### Para Escala (FUTURO):
**🎯 Opção 3: Prometheus + SQLite leve**

- Aproveita infra existente
- Escalável para muitos clusters
- Recording rules otimizadas

---

**Recomendação:** Começar com In-Memory, migrar para Hybrid quando necessário! 🚀

