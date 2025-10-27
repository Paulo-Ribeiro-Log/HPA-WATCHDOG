# Análise: Coleta de Dados para Stress Test

## 📊 Estado Atual (O que JÁ ESTÁ implementado)

### ✅ Infraestrutura Básica

1. **Prometheus Client** (`internal/prometheus/client.go`)
   - ✅ `Query()` - Consultas instantâneas
   - ✅ `QueryRange()` - Consultas de histórico com range de tempo
   - ✅ `GetCPUHistory()` - Histórico de CPU (últimos 5min)
   - ✅ `GetMemoryHistory()` - Histórico de memória (últimos 5min)
   - ✅ `GetReplicaHistory()` - Histórico de réplicas (últimos 5min)
   - ✅ `GetRequestRate()` - Taxa de requisições atual
   - ✅ `GetErrorRate()` - Taxa de erros atual
   - ✅ `GetLatencyP95()` - Latência P95 atual

2. **Persistência SQLite** (`internal/storage/persistence.go`)
   - ✅ Salva snapshots de HPAs (24h de retenção)
   - ✅ Auto-cleanup de dados antigos
   - ✅ Batch inserts otimizados
   - ✅ Schema: `snapshots` table

3. **Modelos de Dados** (`internal/models/stress_test.go`)
   - ✅ `StressTestMetrics` - Estrutura completa
   - ✅ `PeakMetrics` - Campos PRE/PEAK/POST para réplicas
   - ✅ `HPAStressMetrics` - Métricas por HPA
   - ✅ `TimelineEvent` - Linha do tempo
   - ✅ `Recommendation` - Sistema de recomendações

## ✅ O que FOI Implementado (26/10/2025)

### 1. **Captura de Baseline ANTES do Teste** ✅

**Status**: IMPLEMENTADO em `internal/monitor/baseline.go`

**O que foi feito**:
```go
// Antes de iniciar o teste, capturar:
- CPU médio/máximo dos últimos 15-30min
- Memória média/máxima dos últimos 15-30min
- Número de réplicas atual de cada HPA
- Taxa de requisições média
- Taxa de erros baseline
- Latência P95 baseline
```

**Implementação**:
```go
// ✅ IMPLEMENTADO em internal/monitor/baseline.go

type BaselineCollector struct {
    promClient *prometheus.Client
    k8sClient  *K8sClient
}

type BaselineSnapshot struct {
    Timestamp time.Time
    Duration  time.Duration

    // Métricas globais
    TotalHPAs     int
    TotalReplicas int
    CPUAvg, CPUMax, CPUMin, CPUP95 float64
    MemoryAvg, MemoryMax, MemoryMin, MemoryP95 float64
    ReplicasAvg float64

    // Baselines por HPA
    HPABaselines map[string]*HPABaseline
}

func (bc *BaselineCollector) CaptureBaseline(ctx context.Context, duration time.Duration) (*BaselineSnapshot, error) {
    // ✅ Busca todos os HPAs de todos os namespaces
    // ✅ Para cada HPA:
    //    - Busca histórico do Prometheus (30min)
    //    - Calcula estatísticas (avg, max, min, P95, stddev)
    //    - Avalia saúde (healthy/unhealthy)
    // ✅ Retorna BaselineSnapshot completo
}
```

**Documentação**: `docs/BASELINE_COLLECTOR.md` (completa com exemplos)

### 2. **Persistência de StressTestMetrics no SQLite**

**Problema**: Atualmente só salva `HPASnapshot`, não salva `StressTestMetrics` completos.

**O que precisa**:
```sql
-- Nova tabela para stress test results
CREATE TABLE stress_test_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_name TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration INTEGER,
    status TEXT,

    -- Baseline (antes do teste)
    baseline_data TEXT, -- JSON com BaselineSnapshot

    -- Peak metrics (durante o teste)
    peak_data TEXT, -- JSON com PeakMetrics

    -- Results (após o teste)
    result_data TEXT, -- JSON com StressTestMetrics completo

    -- Metadata
    total_clusters INTEGER,
    total_hpas INTEGER,
    total_issues INTEGER,
    health_percentage REAL,
    test_result TEXT -- PASS/FAIL
);

-- Relacionamento com snapshots individuais
CREATE TABLE stress_test_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id INTEGER,
    snapshot_time DATETIME,
    snapshot_data TEXT, -- JSON com HPASnapshot
    FOREIGN KEY (test_id) REFERENCES stress_test_results(id)
);
```

### 3. **Comparação Automática Baseline vs Atual**

**Problema**: Não há lógica de comparação automática durante o teste.

**O que precisa**:
```go
// internal/analyzer/stress_comparator.go
type StressComparator struct {
    baseline *BaselineSnapshot
}

func (sc *StressComparator) CompareWithBaseline(current *HPASnapshot) *ComparisonResult {
    return &ComparisonResult{
        CPUDelta: current.CPUCurrent - sc.baseline.CPUAvg,
        CPUDeltaPercent: ((current.CPUCurrent - sc.baseline.CPUAvg) / sc.baseline.CPUAvg) * 100,

        MemoryDelta: current.MemoryCurrent - sc.baseline.MemoryAvg,
        MemoryDeltaPercent: ((current.MemoryCurrent - sc.baseline.MemoryAvg) / sc.baseline.MemoryAvg) * 100,

        ReplicaDelta: int(current.CurrentReplicas - sc.baseline.ReplicasAvg),
        ReplicaDeltaPercent: ((float64(current.CurrentReplicas) - sc.baseline.ReplicasAvg) / sc.baseline.ReplicasAvg) * 100,

        Status: sc.evaluateStatus(), // NORMAL / DEGRADED / CRITICAL
    }
}
```

### 2. **Extensões do Prometheus Client** ✅

**Status**: IMPLEMENTADO em `internal/prometheus/client.go`

**O que foi feito**:
```go
// ✅ IMPLEMENTADO - 6 novos métodos com range customizável

func (c *Client) GetCPUHistoryRange(ctx context.Context, namespace, hpaName string, start, end time.Time) ([]float64, error)
func (c *Client) GetMemoryHistoryRange(ctx context.Context, namespace, hpaName string, start, end time.Time) ([]float64, error)
func (c *Client) GetReplicaHistoryRange(ctx context.Context, namespace, hpaName string, start, end time.Time) ([]int32, error)
func (c *Client) GetRequestRateHistory(ctx context.Context, namespace, service string, start, end time.Time) ([]float64, error)
func (c *Client) GetErrorRateHistory(ctx context.Context, namespace, service string, start, end time.Time) ([]float64, error)
func (c *Client) GetLatencyP95History(ctx context.Context, namespace, service string, start, end time.Time) ([]float64, error)

// Agora suporta qualquer range: 5min, 30min, 1h, 24h, etc.
// Step: 1 minuto (granularidade)
```

**Queries PromQL**: Ver `docs/BASELINE_COLLECTOR.md` para queries completas

## 🎯 Fluxo Proposto: Stress Test com Baseline

### Fase 1: PRE-TEST (Captura de Baseline)
```
1. Usuário inicia stress test no setup
2. Sistema captura baseline do Prometheus:
   - Últimos 30min de métricas
   - Calcula médias, máximos, P95
   - Salva BaselineSnapshot no SQLite
3. Registra snapshot PRE no StressTestMetrics:
   - TotalReplicasPre
   - CPUAvg/Max
   - MemoryAvg/Max
   - ErrorRateBaseline
   - LatencyBaseline
```

### Fase 2: DURANTE O TESTE (Coleta Contínua)
```
1. A cada scan (ex: 30s):
   - Coleta snapshot atual do Prometheus
   - Compara com baseline
   - Detecta anomalias
   - Atualiza métricas de pico se necessário
   - Salva snapshot no SQLite (stress_test_snapshots)

2. Exibe no dashboard:
   - Comparação atual vs baseline
   - Delta % em tempo real
   - Alertas visuais de degradação
```

### Fase 3: POST-TEST (Análise e Persistência)
```
1. Usuário para teste (tecla S):
   - Captura snapshot POST final
   - Calcula deltas PRE vs POST
   - Gera timeline completa
   - Identifica causa raiz
   - Gera recomendações

2. Salva no SQLite:
   - StressTestMetrics completo
   - Todos os snapshots coletados
   - Timeline de eventos
   - Recomendações geradas

3. Gera relatórios:
   - Markdown com análise
   - PDF com gráficos
```

## ✅ O que FOI Implementado (continuação)

### 3. **Persistência de Baselines e StressTestMetrics no SQLite** ✅

**Status**: IMPLEMENTADO em `internal/storage/persistence.go`

**O que foi feito**:
- ✅ Criado schema `stress_test_baselines` table
- ✅ Criado schema `stress_test_results` table
- ✅ Criado schema `stress_test_snapshots` table
- ✅ Criado schema `stress_test_events` table
- ✅ Criado schema `stress_test_recommendations` table
- ✅ Método `SaveBaseline(testID, baseline)`
- ✅ Método `LoadBaseline(testID)`
- ✅ Método `SaveStressTestResult(testID, result)`
- ✅ Método `SaveStressTestSnapshot(testID, snapshot)`
- ✅ Método `SaveStressTestEvent(...)`
- ✅ Método `ListStressTests()`
- ✅ Schema versioning (v2)

**Documentação**: Ver `internal/storage/persistence.go` (linhas 128-743)

### 4. **Comparação Automática Baseline vs Atual** ✅

**Status**: IMPLEMENTADO em `internal/analyzer/stress_comparator.go`

**O que foi feito**:
```go
// ✅ IMPLEMENTADO - StressComparator completo

type StressComparator struct {
    baseline *models.BaselineSnapshot
    config   *ComparatorConfig
}

// Compara snapshot atual com baseline
func (sc *StressComparator) CompareWithBaseline(current *models.HPASnapshot) *ComparisonResult

// Compara múltiplos snapshots
func (sc *StressComparator) CompareMultiple(snapshots []*models.HPASnapshot) []*ComparisonResult

// Gera resumo das comparações
func (sc *StressComparator) GetSummary(results []*ComparisonResult) *ComparisonSummary
```

**Recursos**:
- ✅ Calcula deltas (absolutos e percentuais)
- ✅ Classifica status: NORMAL / DEGRADED / CRITICAL
- ✅ Thresholds configuráveis por métrica
- ✅ Lista de issues detectados
- ✅ Resumo agregado de múltiplas comparações
- ✅ 9/9 testes unitários passando

**Documentação**: `docs/STRESS_COMPARATOR.md` (completa com exemplos)

## ❌ O que AINDA FALTA Implementar

---

## 📋 Checklist de Implementação

### 1. Baseline Collector ✅
- [x] Criar `internal/monitor/baseline.go`
- [x] Struct `BaselineSnapshot`
- [x] Struct `HPABaseline`
- [x] Método `CaptureBaseline(duration)`
- [x] Integrar com Prometheus QueryRange
- [x] Cálculos estatísticos (avg, max, min, P95, stddev)
- [x] Avaliação de saúde (healthy/unhealthy)
- [x] Documentação completa (`docs/BASELINE_COLLECTOR.md`)

### 2. Prometheus Extensions ✅
- [x] `GetCPUHistoryRange(start, end)` - range customizável
- [x] `GetMemoryHistoryRange(start, end)`
- [x] `GetReplicaHistoryRange(start, end)`
- [x] `GetErrorRateHistory(start, end)`
- [x] `GetLatencyP95History(start, end)`
- [x] `GetRequestRateHistory(start, end)`

### 3. Stress Test Persistence ✅
- [x] Criar schema `stress_test_baselines` table
- [x] Criar schema `stress_test_results` table
- [x] Criar schema `stress_test_snapshots` table
- [x] Criar schema `stress_test_events` table
- [x] Criar schema `stress_test_recommendations` table
- [x] Método `SaveBaseline(testID, baseline)`
- [x] Método `LoadBaseline(testID)`
- [x] Método `SaveStressTestResult(testID, result)`
- [x] Método `SaveStressTestSnapshot(testID, snapshot)`
- [x] Método `SaveStressTestEvent(...)`
- [x] Método `ListStressTests()`

### 4. Comparator ✅
- [x] Criar `internal/analyzer/stress_comparator.go`
- [x] Criar `internal/models/baseline.go` (evitar import cycle)
- [x] Struct `ComparisonResult`
- [x] Struct `ComparisonSummary`
- [x] Struct `ComparatorConfig`
- [x] Método `CompareWithBaseline(current)`
- [x] Método `CompareMultiple(snapshots)`
- [x] Método `GetSummary(results)`
- [x] Método `EvaluateStatus()` (NORMAL/DEGRADED/CRITICAL)
- [x] Testes unitários (9/9 passando)
- [x] Documentação completa (`docs/STRESS_COMPARATOR.md`)

### 5. Engine Integration
- [ ] Modificar collector para capturar baseline antes do teste
- [ ] Integrar comparator no loop de scan
- [ ] Salvar snapshots durante o teste
- [ ] Salvar resultado completo ao finalizar

### 6. TUI Updates
- [ ] Exibir comparação baseline no dashboard
- [ ] Mostrar deltas em tempo real
- [ ] Adicionar view de análise pós-teste
- [ ] Histórico de testes anteriores

## 💾 Exemplo de Dados Persistidos

```json
// stress_test_results table (result_data column)
{
  "test_name": "Black Friday Load Test",
  "start_time": "2025-10-26T14:00:00Z",
  "end_time": "2025-10-26T15:30:00Z",
  "duration": "1h30m",
  "status": "completed",

  "baseline": {
    "cpu_avg": 45.2,
    "cpu_max": 68.1,
    "memory_avg": 52.3,
    "memory_max": 71.2,
    "replicas_avg": 5.2,
    "error_rate": 0.02,
    "latency_p95": 125.3
  },

  "peak": {
    "max_cpu": 89.5,
    "max_cpu_hpa": "prod-cluster/payments/payment-api",
    "max_cpu_time": "2025-10-26T14:45:23Z",

    "total_replicas_pre": 124,
    "total_replicas_peak": 387,
    "total_replicas_post": 156,
    "replica_increase": 263,
    "replica_increase_percent": 212.1
  },

  "results": {
    "total_clusters": 3,
    "total_hpas": 24,
    "total_issues": 5,
    "health_percentage": 79.2,
    "test_result": "PASS",

    "critical_issues": [
      {
        "hpa": "prod-cluster/payments/payment-api",
        "type": "MAXED_OUT",
        "severity": "critical",
        "description": "HPA atingiu maxReplicas (10) e CPU ainda em 95%"
      }
    ],

    "recommendations": [
      {
        "priority": "immediate",
        "target": "prod-cluster/payments/payment-api",
        "action": "Aumentar maxReplicas de 10 para 20",
        "rationale": "HPA saturou durante pico de 15min",
        "impact": "Permite scaling adicional durante Black Friday"
      }
    ]
  }
}
```

## 🚀 Próximos Passos Recomendados

1. ✅ **Implementar Baseline Collector** (CONCLUÍDO)
2. ✅ **Estender Prometheus Queries** (CONCLUÍDO)
3. ✅ **Criar Schema SQLite para Stress Tests** (CONCLUÍDO)
4. ✅ **Implementar Comparator** (CONCLUÍDO)
5. ⏭️ **Integrar no Engine** (PRÓXIMO - 3-4 horas)
6. 🔄 **Atualizar TUI** (2-3 horas)

**Progresso**: 4/6 fases concluídas (67%)
**Tempo restante estimado**: 5-7 horas de desenvolvimento

## ✅ Resumo Final - Implementação Completa

**Status Geral**: ✅ **67% IMPLEMENTADO** (26/10/2025)

### ✅ O que JÁ ESTÁ implementado:

1. ✅ **BaselineCollector** (`internal/monitor/baseline.go`)
   - Captura automática de baseline (30min de histórico)
   - Estatísticas completas (avg, max, min, P95, stddev)
   - Avaliação de saúde por HPA
   - Documentação: `docs/BASELINE_COLLECTOR.md`

2. ✅ **Prometheus Extensions** (`internal/prometheus/client.go`)
   - 6 novos métodos com range customizável
   - Suporte a qualquer período: 5min, 30min, 1h, 24h, etc.
   - Step de 1 minuto (granularidade)

3. ✅ **Persistência SQLite** (`internal/storage/persistence.go`)
   - 5 tabelas: baselines, results, snapshots, events, recommendations
   - 7 métodos de persistência: Save/Load/List
   - Schema versioning (v2)
   - Foreign keys e indexes

4. ✅ **StressComparator** (`internal/analyzer/stress_comparator.go`)
   - Comparação baseline vs atual
   - Status: NORMAL / DEGRADED / CRITICAL
   - Thresholds configuráveis
   - 9/9 testes unitários passando
   - Documentação: `docs/STRESS_COMPARATOR.md`

### ⏭️ O que FALTA implementar:

5. ⏭️ **Integração no Engine** (PRÓXIMO)
   - Capturar baseline antes do teste
   - Loop de comparação durante o teste
   - Salvar resultados ao finalizar

6. 🔄 **TUI Updates**
   - Visualização de comparações em tempo real
   - Dashboard com deltas baseline vs atual
   - Histórico de testes anteriores

**Próximo passo**: Integrar BaselineCollector e StressComparator no engine de coleta.
