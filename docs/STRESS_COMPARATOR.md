# StressComparator - Comparação Baseline vs Atual

## 📋 Visão Geral

O **StressComparator** é o componente responsável por comparar snapshots atuais de HPAs com o baseline capturado antes do stress test, detectando degradações e anomalias em tempo real.

**Localização**: `internal/analyzer/stress_comparator.go`
**Testes**: `internal/analyzer/stress_comparator_test.go` (9/9 testes ✅)

## 🎯 Funcionalidades

### 1. Comparação Scan a Scan

- Compara cada snapshot atual com o baseline correspondente
- Calcula deltas absolutos e percentuais para todas as métricas
- Detecta excedentes de thresholds configuráveis
- Classifica o status: **NORMAL**, **DEGRADED** ou **CRITICAL**

### 2. Métricas Comparadas

| Métrica | Delta Absoluto | Delta Percentual | Threshold |
|---------|----------------|------------------|-----------|
| **CPU** | CPUCurrent - CPUBaseline | (Delta / Baseline) × 100 | Degraded: 30%, Critical: 50% |
| **Memória** | MemoryCurrent - MemoryBaseline | (Delta / Baseline) × 100 | Degraded: 30%, Critical: 50% |
| **Réplicas** | ReplicasCurrent - ReplicasBaseline | (Delta / Baseline) × 100 | Degraded: +3, Critical: +5 |
| **Taxa de Erros** | ErrorRateCurrent - ErrorRateBaseline | Delta absoluto | Critical: +5% |
| **Latência P95** | LatencyCurrent - LatencyBaseline | (Delta / Baseline) × 100 | Critical: +100% |

### 3. Status de Comparação

#### NORMAL ✅
- Todas as métricas dentro dos limites esperados
- Variações menores que thresholds de degradação
- HPA operando de forma saudável

#### DEGRADED ⚠️
- Uma ou mais métricas excederam threshold de degradação
- Ainda não atingiu níveis críticos
- Requer atenção mas não é urgente

**Exemplo**: CPU aumentou 35% (threshold degraded: 30%)

#### CRITICAL 🚨
- Uma ou mais métricas excederam threshold crítico
- Degradação significativa detectada
- Requer ação imediata

**Exemplo**: CPU aumentou 60% (threshold critical: 50%)

## 🔧 Componentes

### ComparisonResult

Resultado detalhado da comparação para um único HPA:

```go
type ComparisonResult struct {
    // Identificação
    Cluster   string
    Namespace string
    HPA       string
    Timestamp time.Time

    // Deltas de CPU
    CPUBaseline      float64
    CPUCurrent       float64
    CPUDelta         float64
    CPUDeltaPercent  float64
    CPUExceededLimit bool

    // Deltas de Memória
    MemoryBaseline      float64
    MemoryCurrent       float64
    MemoryDelta         float64
    MemoryDeltaPercent  float64
    MemoryExceededLimit bool

    // Deltas de Réplicas
    ReplicasBaseline      float64
    ReplicasCurrent       int32
    ReplicaDelta          int32
    ReplicaDeltaPercent   float64
    ReplicasExceededLimit bool

    // Métricas de aplicação
    ErrorRateBaseline  float64
    ErrorRateCurrent   float64
    ErrorRateDelta     float64
    ErrorRateIncreased bool

    LatencyBaseline  float64
    LatencyCurrent   float64
    LatencyDelta     float64
    LatencyIncreased bool

    // Status geral
    Status      ComparisonStatus // NORMAL, DEGRADED, CRITICAL
    Issues      []string         // Lista de problemas detectados
    Severity    string           // info, warning, critical
    Description string           // Descrição geral do estado
}
```

### ComparatorConfig

Configuração dos thresholds de detecção:

```go
type ComparatorConfig struct {
    // Thresholds de CPU
    CPUDegradedThreshold float64 // Default: 30.0%
    CPUCriticalThreshold float64 // Default: 50.0%

    // Thresholds de Memória
    MemoryDegradedThreshold float64 // Default: 30.0%
    MemoryCriticalThreshold float64 // Default: 50.0%

    // Thresholds de Réplicas
    ReplicaDegradedDelta int32   // Default: 3 réplicas
    ReplicaCriticalDelta int32   // Default: 5 réplicas

    // Thresholds de Aplicação
    ErrorRateThreshold float64   // Default: 5.0%
    LatencyThreshold   float64   // Default: 100.0%
}
```

**Configuração padrão**:
```go
config := analyzer.DefaultComparatorConfig()
// CPU: 30% degraded, 50% critical
// Memory: 30% degraded, 50% critical
// Replicas: +3 degraded, +5 critical
// ErrorRate: +5% critical
// Latency: +100% critical
```

### ComparisonSummary

Resumo agregado de múltiplas comparações:

```go
type ComparisonSummary struct {
    Timestamp time.Time

    // Contadores
    TotalHPAs        int
    NormalCount      int
    DegradedCount    int
    CriticalCount    int
    HealthPercentage float64

    // Listas de HPAs problemáticos
    CriticalHPAs []string
    DegradedHPAs []string

    // Métricas agregadas
    TotalCPUDelta     float64
    TotalMemoryDelta  float64
    TotalReplicaDelta int
}
```

## 💻 Uso

### Exemplo 1: Comparação Básica

```go
package main

import (
    "context"
    "time"

    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
)

func main() {
    // 1. Capturar baseline antes do teste
    baselineCollector := monitor.NewBaselineCollector(promClient, k8sClient)
    baseline, _ := baselineCollector.CaptureBaseline(context.Background(), 30*time.Minute)

    // 2. Criar comparador
    comparator := analyzer.NewStressComparator(baseline, nil) // nil = config padrão

    // 3. Durante o teste, comparar cada snapshot
    snapshot := collectCurrentSnapshot() // seu método de coleta

    result := comparator.CompareWithBaseline(snapshot)

    // 4. Verificar resultado
    switch result.Status {
    case analyzer.StatusNormal:
        log.Info().Msg("HPA operando normalmente")
    case analyzer.StatusDegraded:
        log.Warn().
            Strs("issues", result.Issues).
            Msg("HPA degradado detectado")
    case analyzer.StatusCritical:
        log.Error().
            Strs("issues", result.Issues).
            Msg("HPA crítico detectado!")
    }
}
```

### Exemplo 2: Comparação em Lote

```go
// Comparar todos os snapshots de uma vez
snapshots := collectAllSnapshots() // []*models.HPASnapshot

results := comparator.CompareMultiple(snapshots)

// Gerar resumo
summary := comparator.GetSummary(results)

fmt.Printf("Saúde geral: %.1f%%\n", summary.HealthPercentage)
fmt.Printf("Normal: %d, Degraded: %d, Critical: %d\n",
    summary.NormalCount, summary.DegradedCount, summary.CriticalCount)

// Listar HPAs críticos
for _, hpa := range summary.CriticalHPAs {
    fmt.Printf("❌ %s\n", hpa)
}
```

### Exemplo 3: Configuração Customizada

```go
// Criar config com thresholds mais sensíveis
config := &analyzer.ComparatorConfig{
    CPUDegradedThreshold:    15.0, // 15% já é degraded (padrão: 30%)
    CPUCriticalThreshold:    30.0, // 30% é critical (padrão: 50%)
    MemoryDegradedThreshold: 20.0,
    MemoryCriticalThreshold: 40.0,
    ReplicaDegradedDelta:    2,    // +2 réplicas (padrão: 3)
    ReplicaCriticalDelta:    4,    // +4 réplicas (padrão: 5)
    ErrorRateThreshold:      3.0,  // +3% de erro (padrão: 5%)
    LatencyThreshold:        50.0, // +50% de latência (padrão: 100%)
}

comparator := analyzer.NewStressComparator(baseline, config)
```

## 📊 Exemplos de Output

### ComparisonResult - Status Normal

```json
{
  "cluster": "prod-cluster",
  "namespace": "payments",
  "hpa": "payment-api",
  "status": "NORMAL",
  "severity": "info",
  "description": "HPA está operando dentro dos limites esperados",
  "cpu_baseline": 50.0,
  "cpu_current": 52.0,
  "cpu_delta": 2.0,
  "cpu_delta_percent": 4.0,
  "memory_baseline": 45.0,
  "memory_current": 46.0,
  "memory_delta": 1.0,
  "memory_delta_percent": 2.2,
  "replicas_baseline": 3.0,
  "replicas_current": 3,
  "replica_delta": 0,
  "issues": []
}
```

### ComparisonResult - Status Degraded

```json
{
  "cluster": "prod-cluster",
  "namespace": "payments",
  "hpa": "payment-api",
  "status": "DEGRADED",
  "severity": "warning",
  "description": "1 degradações detectadas durante stress test",
  "cpu_baseline": 50.0,
  "cpu_current": 67.5,
  "cpu_delta": 17.5,
  "cpu_delta_percent": 35.0,
  "cpu_exceeded_limit": true,
  "issues": [
    "CPU aumentou 35.0% (de 50.0% para 67.5%)"
  ]
}
```

### ComparisonResult - Status Critical

```json
{
  "cluster": "prod-cluster",
  "namespace": "payments",
  "hpa": "payment-api",
  "status": "CRITICAL",
  "severity": "critical",
  "description": "5 problemas críticos detectados durante stress test",
  "cpu_baseline": 50.0,
  "cpu_current": 80.0,
  "cpu_delta": 30.0,
  "cpu_delta_percent": 60.0,
  "cpu_exceeded_limit": true,
  "memory_baseline": 45.0,
  "memory_current": 70.0,
  "memory_delta": 25.0,
  "memory_delta_percent": 55.6,
  "memory_exceeded_limit": true,
  "replicas_baseline": 3.0,
  "replicas_current": 9,
  "replica_delta": 6,
  "replica_delta_percent": 200.0,
  "replicas_exceeded_limit": true,
  "error_rate_baseline": 0.5,
  "error_rate_current": 6.0,
  "error_rate_delta": 5.5,
  "error_rate_increased": true,
  "latency_baseline": 100.0,
  "latency_current": 250.0,
  "latency_delta": 150.0,
  "latency_increased": true,
  "issues": [
    "CPU aumentou 60.0% (de 50.0% para 80.0%)",
    "Memória aumentou 55.6% (de 45.0% para 70.0%)",
    "Réplicas aumentaram em 6 (de 3 para 9)",
    "Taxa de erros aumentou 5.50% (de 0.50% para 6.00%)",
    "Latência P95 aumentou 150.0% (de 100.0ms para 250.0ms)"
  ]
}
```

### ComparisonSummary

```go
summary.String()
// Output: "Total: 24 HPAs | Normal: 18 | Degraded: 4 | Critical: 2 | Saúde: 75.0%"
```

## 🧪 Testes

**Localização**: `internal/analyzer/stress_comparator_test.go`

### Cobertura de Testes (9/9 ✅)

| # | Teste | Descrição |
|---|-------|-----------|
| 1 | `TestStressComparator_CompareWithBaseline_Normal` | Snapshot normal (sem mudanças significativas) |
| 2 | `TestStressComparator_CompareWithBaseline_Degraded` | CPU degradada (+35%) |
| 3 | `TestStressComparator_CompareWithBaseline_Critical` | Múltiplas métricas críticas |
| 4 | `TestStressComparator_CompareWithBaseline_HPANotInBaseline` | HPA novo (não estava no baseline) |
| 5 | `TestStressComparator_CompareMultiple` | Comparação em lote |
| 6 | `TestStressComparator_GetSummary` | Geração de resumo |
| 7 | `TestStressComparator_CustomConfig` | Configuração customizada |
| 8 | `TestComparatorConfig_Default` | Valores padrão da config |
| 9 | `TestComparisonSummary_String` | Representação em string do resumo |

### Executar Testes

```bash
# Executar testes do StressComparator
go test -v ./internal/analyzer/stress_comparator_test.go ./internal/analyzer/stress_comparator.go

# Executar com cobertura
go test -cover ./internal/analyzer/stress_comparator_test.go ./internal/analyzer/stress_comparator.go
```

## 🔄 Integração com Outros Componentes

### 1. BaselineCollector

```go
// Captura baseline antes do teste
baseline, _ := baselineCollector.CaptureBaseline(ctx, 30*time.Minute)

// Cria comparador com baseline
comparator := analyzer.NewStressComparator(baseline, nil)
```

### 2. HPASnapshot

```go
// Coleta snapshot atual
snapshot := collector.CollectSnapshot(ctx, hpa)

// Compara com baseline
result := comparator.CompareWithBaseline(snapshot)
```

### 3. StressTestMetrics

```go
// Durante o teste, acumula resultados
results = append(results, comparator.CompareWithBaseline(snapshot))

// Ao finalizar, gera resumo
summary := comparator.GetSummary(results)

// Salva no StressTestMetrics
stressMetrics.HealthPercentage = summary.HealthPercentage
stressMetrics.TotalHPAsWithIssues = summary.DegradedCount + summary.CriticalCount
```

## 🎨 Visualização na TUI

Durante o stress test, o dashboard exibe comparações em tempo real:

```
╭─ Stress Test Dashboard ──────────────────────────────────╮
│                                                           │
│  HPA: prod-cluster/payments/payment-api                  │
│                                                           │
│  Status: DEGRADED ⚠️                                      │
│                                                           │
│  ┌─ CPU ────────────────────┐  ┌─ Memória ──────────────┐│
│  │ Baseline:   50.0%        │  │ Baseline:   45.0%      ││
│  │ Atual:      67.5%        │  │ Atual:      48.0%      ││
│  │ Delta:     +17.5% (+35%) │  │ Delta:      +3.0% (+7%)││
│  │ Status:     ⚠️ DEGRADED  │  │ Status:     ✅ NORMAL  ││
│  └──────────────────────────┘  └────────────────────────┘│
│                                                           │
│  Issues:                                                  │
│  • CPU aumentou 35.0% (de 50.0% para 67.5%)              │
│                                                           │
╰───────────────────────────────────────────────────────────╯
```

## ⚙️ Configuração Avançada

### Thresholds Personalizados por Ambiente

```go
// Produção - thresholds rigorosos
prodConfig := &analyzer.ComparatorConfig{
    CPUDegradedThreshold: 20.0,
    CPUCriticalThreshold: 40.0,
    // ...
}

// Staging - thresholds relaxados
stagingConfig := &analyzer.ComparatorConfig{
    CPUDegradedThreshold: 50.0,
    CPUCriticalThreshold: 80.0,
    // ...
}
```

### Filtrar apenas HPAs Críticos

```go
results := comparator.CompareMultiple(snapshots)

criticalResults := []analyzer.ComparisonResult{}
for _, result := range results {
    if result.Status == analyzer.StatusCritical {
        criticalResults = append(criticalResults, result)
    }
}
```

## 📝 Próximos Passos

1. ✅ **StressComparator implementado**
2. ⏭️ **Integrar no Engine** (próxima tarefa)
3. ⏭️ Adicionar visualização na TUI
4. ⏭️ Implementar alertas em tempo real
5. ⏭️ Gerar relatórios com comparações

## 🐛 Troubleshooting

### HPA não encontrado no baseline

**Problema**: `ComparisonResult` retorna status NORMAL com issue "HPA não encontrado no baseline"

**Causa**: HPA foi criado após captura do baseline

**Solução**: Re-capturar baseline antes do teste ou filtrar HPAs novos

### Deltas muito sensíveis

**Problema**: Muitos falsos positivos (HPAs marcados como degraded)

**Causa**: Thresholds padrão muito baixos para seu ambiente

**Solução**: Ajustar thresholds usando `ComparatorConfig` customizado

### Deltas não detectados

**Problema**: Degradações reais não sendo detectadas

**Causa**: Thresholds muito altos

**Solução**: Reduzir thresholds ou verificar se baseline foi capturado corretamente

## ✅ Resumo

- ✅ **Implementado**: StressComparator completo
- ✅ **Testes**: 9/9 testes passando
- ✅ **Documentação**: Completa
- ✅ **Integração**: Usa `models.BaselineSnapshot` (sem import cycle)
- ⏭️ **Próximo**: Integrar no Engine para uso em stress tests
