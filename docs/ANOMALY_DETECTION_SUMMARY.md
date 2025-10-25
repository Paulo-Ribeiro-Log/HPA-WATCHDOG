# Detecção de Anomalias - Sumário Executivo

Referência rápida das anomalias detectadas pelo HPA Watchdog.

## 🎯 Quick Reference

### Fase 1 - MVP (Implementar AGORA)

| # | Anomalia | Severidade | Detecção | Limite |
|---|----------|------------|----------|--------|
| 1 | **Oscillation** | 🔴 Critical | >5 mudanças réplicas | 5min |
| 2 | **Maxed Out** | 🔴 Critical | replicas=max + CPU>target+20% | 2min |
| 3 | **OOMKilled** | 🔴 Critical | Pod killed por OOM | Imediato |
| 4 | **Pods Not Ready** | 🔴 Critical | <70% pods ready | 3min |
| 5 | **High Error Rate** | 🔴 Critical | >5% erros 5xx | 2min |

### Fase 2 - Expansão

| # | Anomalia | Severidade | Detecção | Limite |
|---|----------|------------|----------|--------|
| 6 | **Scaling Stuck** | 🔴 Critical | CPU>target+30% sem escalar | 5min |
| 7 | **CPU Throttling** | 🔴 Critical | >25% throttling | 5min |
| 8 | **High Latency** | 🔴 Critical | P95>1000ms ou 2x baseline | 3min |
| 9 | **Underutilization** | 🟡 Warning | CPU<target-40% + >3 replicas | 15min |
| 10 | **CrashLoopBackOff** | 🔴 Critical | Pod em crash loop | Imediato |

## 📊 Decision Matrix

```
Problema Detectado → Ação Sugerida

┌─────────────────────────────────────────────────────────────┐
│ SCALING                                                      │
├─────────────────────────────────────────────────────────────┤
│ Oscillation        → Aumentar stabilizationWindow           │
│ Maxed Out          → Aumentar maxReplicas                   │
│ Scaling Stuck      → Verificar quota/capacidade cluster     │
│ Underutilization   → Reduzir maxReplicas ou target          │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ POD HEALTH                                                   │
├─────────────────────────────────────────────────────────────┤
│ OOMKilled          → Aumentar memory limit                  │
│ CrashLoopBackOff   → Verificar logs + dependências          │
│ Pods Not Ready     → Ajustar readiness probe                │
│ High Restart Rate  → Investigar causa (OOM? probe?)         │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ PERFORMANCE                                                  │
├─────────────────────────────────────────────────────────────┤
│ CPU Throttling     → Remover CPU limit ou aumentar          │
│ High Error Rate    → Scale up ou verificar dependências     │
│ High Latency       → Scale up ou profile aplicação          │
└─────────────────────────────────────────────────────────────┘
```

## 🚨 Alert Priority

```
P0 (Imediato):
  ├─ OOMKilled
  ├─ CrashLoopBackOff
  └─ High Error Rate (>10%)

P1 (Urgente - <5min):
  ├─ Maxed Out
  ├─ Pods Not Ready
  ├─ High Error Rate (>5%)
  └─ Scaling Stuck

P2 (Importante - <15min):
  ├─ Oscillation
  ├─ CPU Throttling
  └─ High Latency

P3 (Atenção - <1h):
  ├─ Underutilization
  ├─ Frequent Scaling
  └─ Config Issues
```

## 💡 Common Correlations

```
1. HPA maxed + CPU high + Latency high
   → Capacity issue, need to scale

2. HPA scaled up + Pods not ready
   → Readiness probe or dependencies issue

3. HPA scaled up + Pods OOMKilled
   → Memory limit too low or leak

4. CPU usage low + CPU throttling high
   → CPU limit too low (remove limit!)

5. Oscillation + Frequent scaling
   → HPA config too sensitive
```

## 📈 Baseline Values

### Thresholds Recomendados

```yaml
# Scaling
oscillation_max_changes: 5 (5min)
maxed_out_deviation: 20% (2min)
stuck_deviation: 30% (5min)
underutilization_deviation: 40% (15min)

# Pod Health
crash_loop_max_restarts: 5 (10min)
not_ready_threshold: 70% (3min)
restart_rate: 2 (15min)

# Performance
cpu_throttling: 25% (5min)
error_rate: 5% (2min)
latency_p95: 1000ms (3min)
```

### HPA Config Ideal

```yaml
# Good defaults
minReplicas: 2
maxReplicas: 10
cpuTarget: 70%
memoryTarget: 80%

stabilizationWindow:
  scaleDown: 300s  # 5min
  scaleUp: 0s      # Imediato

# Resources (per pod)
requests:
  cpu: 500m
  memory: 512Mi
limits:
  # cpu: NONE (evita throttling)
  memory: 1Gi  # 2x request
```

## 🔄 Alert Lifecycle

```
1. DETECT (scan_interval: 30s)
   ├─ Scan HPAs + Deployments
   ├─ Apply detection rules
   └─ Check thresholds + duration

2. VALIDATE (time-based)
   ├─ Confirm problem persists
   ├─ Duration > threshold?
   └─ Not in cooldown period?

3. ENRICH
   ├─ Add metrics context
   ├─ Add K8s events
   ├─ Correlate related alerts
   └─ Suggest actions

4. ALERT
   ├─ Create UnifiedAlert
   ├─ Set severity
   ├─ Add to dashboard
   └─ (Future) Send notification

5. COOLDOWN
   ├─ Start cooldown timer
   ├─ Prevent duplicate alerts
   └─ Auto-ack if resolved
```

## 📁 File Structure

```
internal/analyzer/
├── detector.go              # Engine principal
├── detector_test.go         # Testes
├── models.go                # Anomaly, Alert structs
├── config.go                # Thresholds config
├── rules/                   # Regras de detecção
│   ├── scaling/
│   │   ├── oscillation.go
│   │   ├── maxed_out.go
│   │   ├── stuck.go
│   │   └── underutilization.go
│   ├── pods/
│   │   ├── oom_killed.go
│   │   ├── crash_loop.go
│   │   ├── not_ready.go
│   │   └── restart_rate.go
│   └── performance/
│       ├── cpu_throttling.go
│       ├── error_rate.go
│       └── latency.go
├── correlation.go           # Alert correlation
└── enrichment.go            # Context enrichment
```

## 🧪 Testing Strategy

```bash
# Unit tests (cada regra)
go test ./internal/analyzer/rules/... -v

# Integration tests (detector completo)
go test ./internal/analyzer/... -v

# Real cluster test
./build/hpa-watchdog test \
  --cluster production \
  --namespace default \
  --prometheus

# Simulate anomaly
kubectl apply -f tests/fixtures/oscillating-hpa.yaml
# Wait 5min
# Verify alert detected
```

## 📚 Links Úteis

- [Documentação Completa](./ANOMALY_DETECTION.md)
- [Guia de Testes](./TESTING.md)
- [Configuração](../configs/watchdog.yaml)
- [CLAUDE.md](../CLAUDE.md)

---

**Quick Start:**

1. Ler [ANOMALY_DETECTION.md](./ANOMALY_DETECTION.md) completo
2. Implementar Fase 1 (5 anomalias MVP)
3. Testar em cluster staging
4. Deploy em produção
5. Iterar com Fase 2

**Dúvidas?** Consulte a documentação completa ou código implementado.
