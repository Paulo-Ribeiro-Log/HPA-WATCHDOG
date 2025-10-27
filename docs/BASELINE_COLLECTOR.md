# BaselineCollector - Captura de Estado Pré-Teste

## 📋 Visão Geral

O **BaselineCollector** é responsável por capturar o estado completo do cluster ANTES de iniciar um stress test. Ele busca dados históricos do Prometheus (ex: últimos 30 minutos) e calcula estatísticas que servirão como **baseline** para comparação durante e após o teste.

## 🎯 Por que Baseline?

Sem baseline, não é possível responder perguntas fundamentais:

- ❓ O HPA está escalando **mais** do que antes?
- ❓ A CPU está **acima** do normal?
- ❓ O sistema estava **saudável** antes do teste?
- ❓ Quanto o stress test **impactou** as métricas?

**Com baseline**:
- ✅ "CPU subiu de 45% → 89% (+44 pontos, +97%)"
- ✅ "Réplicas aumentaram de 5 → 12 (+7, +140%)"
- ✅ "HPA estava saudável antes do teste (CPU=45%, sem oscilação)"

## 📦 Componentes

### 1. BaselineSnapshot

Captura o **estado global** do cluster:

```go
type BaselineSnapshot struct {
    Timestamp time.Time
    Duration  time.Duration // Ex: 30min

    // Métricas globais
    TotalClusters int
    TotalHPAs     int
    TotalReplicas int

    // Estatísticas de CPU
    CPUAvg    float64
    CPUMax    float64
    CPUMin    float64
    CPUP95    float64

    // Estatísticas de Memória
    MemoryAvg float64
    MemoryMax float64
    MemoryMin float64
    MemoryP95 float64

    // Estatísticas de Réplicas
    ReplicasAvg float64
    ReplicasMax int32
    ReplicasMin int32

    // Tráfego
    RequestRateAvg float64
    ErrorRateAvg   float64
    LatencyP95Avg  float64

    // Baselines por HPA
    HPABaselines map[string]*HPABaseline
}
```

### 2. HPABaseline

Baseline **individual** de cada HPA:

```go
type HPABaseline struct {
    // Identificação
    Cluster   string
    Namespace string
    Name      string

    // Configuração
    MinReplicas int32
    MaxReplicas int32
    TargetCPU   int32
    CurrentReplicas int32

    // Estatísticas do período (ex: 30min)
    CPUAvg    float64
    CPUMax    float64
    CPUMin    float64

    MemoryAvg float64
    MemoryMax float64
    MemoryMin float64

    ReplicasAvg    float64
    ReplicasMax    int32
    ReplicasMin    int32
    ReplicasStdDev float64 // Desvio padrão (oscilação)

    // Métricas de aplicação
    RequestRateAvg float64
    ErrorRateAvg   float64
    LatencyP95Avg  float64

    // Avaliação
    Timestamp time.Time
    Healthy   bool   // Se estava saudável
    Notes     string // Observações
}
```

### 3. BaselineCollector

Orquestrador da coleta:

```go
type BaselineCollector struct {
    promClient *prometheus.Client
    k8sClient  *K8sClient
}
```

## 🚀 Como Usar

### Exemplo Básico

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/monitor"
    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/prometheus"
)

func main() {
    ctx := context.Background()

    // Clientes
    promClient, _ := prometheus.NewClient("prod-cluster", "http://prometheus:9090")
    k8sClient, _ := monitor.NewK8sClient("prod-cluster", nil)

    // Cria baseline collector
    collector := monitor.NewBaselineCollector(promClient, k8sClient)

    // Captura baseline dos últimos 30 minutos
    baseline, err := collector.CaptureBaseline(ctx, 30*time.Minute)
    if err != nil {
        panic(err)
    }

    // Exibe resumo global
    fmt.Printf("Baseline capturado em %s\n", baseline.Timestamp)
    fmt.Printf("Período analisado: %s\n", baseline.Duration)
    fmt.Printf("Total de HPAs: %d\n", baseline.TotalHPAs)
    fmt.Printf("Total de réplicas: %d\n", baseline.TotalReplicas)
    fmt.Printf("CPU média: %.1f%%\n", baseline.CPUAvg)
    fmt.Printf("CPU máxima: %.1f%%\n", baseline.CPUMax)
    fmt.Printf("Memória média: %.1f%%\n", baseline.MemoryAvg)

    // Analisa cada HPA
    for hpaKey, hpaBaseline := range baseline.HPABaselines {
        fmt.Printf("\n=== %s ===\n", hpaKey)
        fmt.Printf("  CPU: avg=%.1f%%, max=%.1f%%\n",
            hpaBaseline.CPUAvg, hpaBaseline.CPUMax)
        fmt.Printf("  Réplicas: atual=%d, avg=%.1f, max=%d\n",
            hpaBaseline.CurrentReplicas,
            hpaBaseline.ReplicasAvg,
            hpaBaseline.ReplicasMax)
        fmt.Printf("  Oscilação: stddev=%.2f\n", hpaBaseline.ReplicasStdDev)
        fmt.Printf("  Saudável: %v\n", hpaBaseline.Healthy)
        if !hpaBaseline.Healthy {
            fmt.Printf("  ⚠️  %s\n", hpaBaseline.Notes)
        }
    }
}
```

### Exemplo com Comparação Durante Stress Test

```go
// ANTES do teste
baseline, _ := collector.CaptureBaseline(ctx, 30*time.Minute)

// Salva baseline
initialCPU := baseline.HPABaselines["prod/payments/payment-api"].CPUAvg
initialReplicas := baseline.HPABaselines["prod/payments/payment-api"].CurrentReplicas

fmt.Printf("Baseline: CPU=%.1f%%, Replicas=%d\n", initialCPU, initialReplicas)

// === INICIA STRESS TEST ===
time.Sleep(10 * time.Minute)

// DURANTE o teste
currentSnapshot, _ := k8sClient.CollectHPASnapshot(ctx, hpa)

// Compara com baseline
cpuDelta := currentSnapshot.CPUCurrent - initialCPU
cpuDeltaPercent := (cpuDelta / initialCPU) * 100

replicaDelta := int(currentSnapshot.CurrentReplicas) - int(initialReplicas)
replicaDeltaPercent := (float64(replicaDelta) / float64(initialReplicas)) * 100

fmt.Printf("DURANTE teste:\n")
fmt.Printf("  CPU: %.1f%% → %.1f%% (%+.1f, %+.1f%%)\n",
    initialCPU, currentSnapshot.CPUCurrent, cpuDelta, cpuDeltaPercent)
fmt.Printf("  Replicas: %d → %d (%+d, %+.1f%%)\n",
    initialReplicas, currentSnapshot.CurrentReplicas, replicaDelta, replicaDeltaPercent)

if cpuDeltaPercent > 50 {
    fmt.Printf("⚠️  CPU aumentou mais de 50%%!\n")
}
```

## 📊 Métricas Coletadas

### Do Prometheus (Range Queries)

O BaselineCollector usa os novos métodos do `prometheus.Client`:

```go
// CPU histórico (30min)
cpuHistory, _ := promClient.GetCPUHistoryRange(ctx, namespace, hpaName, start, end)
// Retorna: []float64{45.2, 47.1, 46.8, ...}

// Memória histórico
memHistory, _ := promClient.GetMemoryHistoryRange(ctx, namespace, hpaName, start, end)

// Réplicas histórico
replicaHistory, _ := promClient.GetReplicaHistoryRange(ctx, namespace, hpaName, start, end)
// Retorna: []int32{5, 5, 6, 5, 5, ...}

// Request rate histórico
reqRate, _ := promClient.GetRequestRateHistory(ctx, namespace, service, start, end)

// Error rate histórico
errRate, _ := promClient.GetErrorRateHistory(ctx, namespace, service, start, end)

// Latency P95 histórico
latency, _ := promClient.GetLatencyP95History(ctx, namespace, service, start, end)
```

### Queries PromQL Usadas

**CPU**:
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="payments",pod=~"payment-api.*"}[1m])) /
sum(kube_pod_container_resource_requests{namespace="payments",pod=~"payment-api.*",resource="cpu"}) * 100
```

**Memória**:
```promql
sum(container_memory_working_set_bytes{namespace="payments",pod=~"payment-api.*"}) /
sum(kube_pod_container_resource_requests{namespace="payments",pod=~"payment-api.*",resource="memory"}) * 100
```

**Réplicas**:
```promql
kube_horizontalpodautoscaler_status_current_replicas{namespace="payments",horizontalpodautoscaler="payment-api"}
```

**Request Rate**:
```promql
sum(rate(http_requests_total{namespace="payments",service="payment-api"}[1m]))
```

**Error Rate**:
```promql
sum(rate(http_requests_total{namespace="payments",service="payment-api",status=~"5.."}[1m])) /
sum(rate(http_requests_total{namespace="payments",service="payment-api"}[1m])) * 100
```

**Latency P95**:
```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{namespace="payments",service="payment-api"}[5m])) by (le)) * 1000
```

## 🔍 Avaliação de Saúde

O BaselineCollector avalia automaticamente se cada HPA estava **saudável** antes do teste:

### Critérios de HPA Não Saudável

```go
func (bc *BaselineCollector) evaluateHPAHealth(baseline *HPABaseline, hpa *models.HPASnapshot) bool {
    // 1. No limite com CPU alta
    if baseline.CurrentReplicas >= baseline.MaxReplicas && baseline.CPUAvg > 80 {
        baseline.Notes = "HPA estava no limite com CPU alta durante baseline"
        return false
    }

    // 2. CPU consistentemente muito alta
    if baseline.CPUAvg > 85 {
        baseline.Notes = "CPU muito alta durante baseline"
        return false
    }

    // 3. Oscilação excessiva de réplicas
    if baseline.ReplicasStdDev > 2.0 {
        baseline.Notes = "Oscilação excessiva de réplicas durante baseline"
        return false
    }

    // 4. Taxa de erros alta
    if baseline.ErrorRateAvg > 1.0 { // > 1%
        baseline.Notes = "Taxa de erros alta durante baseline"
        return false
    }

    return true
}
```

### Exemplos

**HPA Saudável** ✅:
```
prod/payments/payment-api:
  CPU: avg=45.2%, max=68.1%
  Réplicas: atual=5, avg=5.2, stddev=0.8
  Taxa de erros: 0.02%
  Healthy: true
```

**HPA Não Saudável** ❌:
```
prod/checkout/checkout-api:
  CPU: avg=87.3%, max=95.2%
  Réplicas: atual=10, max=10 (NO LIMITE)
  Taxa de erros: 0.15%
  Healthy: false
  ⚠️  HPA estava no limite com CPU alta durante baseline
```

## 📈 Cálculos Estatísticos

### Média
```go
func calculateAvg(values []float64) float64 {
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    return sum / float64(len(values))
}
```

### Percentil 95
```go
func calculateP95(values []float64) float64 {
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)

    idx := int(float64(len(sorted)) * 0.95)
    if idx >= len(sorted) {
        idx = len(sorted) - 1
    }
    return sorted[idx]
}
```

### Desvio Padrão (Oscilação)
```go
func calculateStdDev(values []float64, avg float64) float64 {
    variance := 0.0
    for _, v := range values {
        diff := v - avg
        variance += diff * diff
    }
    variance /= float64(len(values))
    return math.Sqrt(variance)
}
```

**Interpretação do StdDev**:
- `< 1.0` - Estável
- `1.0 - 2.0` - Oscilação moderada
- `> 2.0` - Oscilação excessiva (problema!)

## 💾 Persistência

O baseline deve ser salvo no SQLite antes de iniciar o teste:

```go
// Captura baseline
baseline, _ := collector.CaptureBaseline(ctx, 30*time.Minute)

// Salva no SQLite (TODO: implementar)
err := persistence.SaveBaseline(baseline)

// Inicia stress test
stressTest := NewStressTest(baseline)
stressTest.Start()
```

## 🔄 Fluxo Completo

```
1. CAPTURA BASELINE (30min antes)
   ↓
   BaselineCollector.CaptureBaseline()
   ↓
   - Lista todos os HPAs
   - Para cada HPA:
     ↓
     a) Busca histórico Prometheus (30min)
     b) Calcula estatísticas (avg, max, min, P95, stddev)
     c) Avalia saúde (healthy/unhealthy)
     d) Cria HPABaseline
   ↓
   - Calcula estatísticas globais
   - Retorna BaselineSnapshot completo

2. SALVA BASELINE
   ↓
   SQLite: stress_test_baselines table

3. INICIA STRESS TEST
   ↓
   Durante teste: compara cada snapshot com baseline
   ↓
   - CPU atual vs CPU baseline → Delta
   - Réplicas atual vs Réplicas baseline → Delta
   - Detecta anomalias

4. FINALIZA TESTE
   ↓
   Gera relatório:
   - Baseline PRE
   - Métricas PEAK durante
   - Estado POST
   - Deltas e análise
```

## 📝 Exemplo de Output

```
=== BASELINE CAPTURADO ===
Timestamp: 2025-10-26 14:00:00
Período: 30min
Clusters: 1
HPAs: 24
Réplicas totais: 124

Estatísticas Globais:
  CPU: avg=45.2%, max=68.1%, min=12.3%, P95=62.5%
  Memória: avg=52.3%, max=71.2%, min=18.7%, P95=67.8%
  Réplicas: avg=5.2, max=12, min=2

HPAs Saudáveis: 22/24 (91.7%)

HPAs com Problemas:
  ❌ prod/checkout/checkout-api
     Motivo: HPA estava no limite com CPU alta durante baseline
     CPU: avg=87.3%, Réplicas: 10/10 (maxReplicas)

  ❌ dev/legacy/old-service
     Motivo: Oscilação excessiva de réplicas durante baseline
     Réplicas: stddev=3.2
```

## 🎯 Casos de Uso

### 1. Validar Estado Pré-Teste
```go
baseline, _ := collector.CaptureBaseline(ctx, 30*time.Minute)

unhealthyCount := 0
for _, hpa := range baseline.HPABaselines {
    if !hpa.Healthy {
        unhealthyCount++
        fmt.Printf("⚠️  %s/%s não está saudável: %s\n",
            hpa.Namespace, hpa.Name, hpa.Notes)
    }
}

if unhealthyCount > 0 {
    fmt.Printf("\n❌ ATENÇÃO: %d HPAs não estão saudáveis!\n", unhealthyCount)
    fmt.Printf("Recomenda-se corrigir antes de iniciar o stress test.\n")
    return
}

fmt.Println("✅ Todos os HPAs estão saudáveis. Pode iniciar o teste!")
```

### 2. Comparação em Tempo Real
```go
// Durante o teste
for snapshot := range snapshotChan {
    hpaKey := fmt.Sprintf("%s/%s/%s", snapshot.Cluster, snapshot.Namespace, snapshot.Name)
    baseline := baselineSnapshot.HPABaselines[hpaKey]

    cpuDelta := snapshot.CPUCurrent - baseline.CPUAvg
    if cpuDelta > 30 {
        fmt.Printf("⚠️  CPU spike: %.1f%% → %.1f%% (%+.1f%%)\n",
            baseline.CPUAvg, snapshot.CPUCurrent, cpuDelta)
    }
}
```

### 3. Análise Pós-Teste
```go
// Após o teste
report := GenerateStressTestReport(baseline, peakMetrics, postSnapshot)

fmt.Printf("=== RELATÓRIO DE STRESS TEST ===\n\n")
fmt.Printf("PRE (baseline):\n")
fmt.Printf("  CPU: %.1f%%\n", baseline.CPUAvg)
fmt.Printf("  Réplicas: %d\n", baseline.TotalReplicas)
fmt.Printf("\n")

fmt.Printf("PEAK (durante teste):\n")
fmt.Printf("  CPU: %.1f%% (%+.1f%%)\n", peakMetrics.MaxCPUPercent,
    peakMetrics.MaxCPUPercent - baseline.CPUAvg)
fmt.Printf("  Réplicas: %d (%+d)\n", peakMetrics.TotalReplicasPeak,
    peakMetrics.TotalReplicasPeak - baseline.TotalReplicas)
fmt.Printf("\n")

fmt.Printf("POST (após teste):\n")
fmt.Printf("  CPU: %.1f%% (%+.1f%% do baseline)\n", postSnapshot.CPUAvg,
    postSnapshot.CPUAvg - baseline.CPUAvg)
fmt.Printf("  Réplicas: %d (%+d do baseline)\n", postSnapshot.TotalReplicas,
    postSnapshot.TotalReplicas - baseline.TotalReplicas)
```

## 🚨 Troubleshooting

### Erro: "Prometheus client not connected"
```go
// Solução: Testar conexão antes
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := promClient.TestConnection(ctx); err != nil {
    log.Fatal("Prometheus não acessível:", err)
}
```

### Erro: "No data available"
```go
// Possíveis causas:
// 1. Prometheus não tem dados históricos suficientes
// 2. Labels/queries não batem com o ambiente
// 3. Métricas não estão sendo coletadas

// Solução: Verificar query manualmente
query := `kube_horizontalpodautoscaler_status_current_replicas{namespace="payments"}`
result, _ := promClient.Query(ctx, query)
fmt.Printf("Resultado: %+v\n", result)
```

### Aviso: "Failed to get [metric] history"
```go
// Não é erro fatal - algumas métricas são opcionais
// BaselineCollector usa fallback para valores atuais

// Métricas essenciais (sempre necessárias):
// - CPU
// - Memória
// - Réplicas

// Métricas opcionais (podem falhar):
// - Request Rate
// - Error Rate
// - Latency
```

## 📚 Referências

- **Arquivo**: `internal/monitor/baseline.go`
- **Prometheus Client**: `internal/prometheus/client.go`
- **Modelos**: `internal/models/types.go`, `internal/models/stress_test.go`
- **Testes**: `internal/monitor/baseline_test.go` (TODO)
