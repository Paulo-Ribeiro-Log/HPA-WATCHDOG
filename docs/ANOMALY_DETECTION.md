# Detecção de Anomalias - HPA Watchdog

Estratégia completa de detecção de comportamentos anormais em HPAs e Deployments.

## 📋 Índice

- [Filosofia](#filosofia)
- [Categorias de Anomalias](#categorias-de-anomalias)
  - [1. Anomalias de Escalonamento](#1-anomalias-de-escalonamento-hpa-behavior)
  - [2. Anomalias de Deployment/Pods](#2-anomalias-de-deploymentpods-application-health)
  - [3. Anomalias de Métricas](#3-anomalias-de-métricas-performance)
  - [4. Anomalias de Configuração](#4-anomalias-de-configuração-config-issues)
- [Limites e Thresholds](#limites-e-thresholds-recomendados)
- [Roadmap de Implementação](#roadmap-de-implementação)
- [Ideias Futuras](#ideias-futuras)

---

## Filosofia

### **Signal vs Noise**

O objetivo é **alertar apenas o que importa** - problemas reais que precisam de ação.

**Princípios:**
- ✅ **Actionable alerts** - Cada alerta deve sugerir uma ação clara
- ❌ **Evitar alert fatigue** - Muitos alertas = todos ignorados
- 🎯 **Root cause focus** - Identificar a causa raiz, não sintomas
- ⏱️ **Time-based validation** - Confirmar problema antes de alertar
- 🔕 **Cooldown periods** - Não spam de alertas repetidos

### **Níveis de Severidade**

```
🔴 CRITICAL - Requer ação imediata
   Exemplos: OOMKilled, CrashLoop, Maxed Out

🟡 WARNING - Requer atenção mas não urgente
   Exemplos: Underutilization, Inefficient Config

🔵 INFO - Informativo, sem ação necessária
   Exemplos: Config changes, Scale events
```

---

## Categorias de Anomalias

## 1️⃣ Anomalias de Escalonamento (HPA Behavior)

### 🔴 CRÍTICAS

#### A) Thrashing / Oscillation

**Descrição:** HPA escalando up/down rapidamente (comportamento instável)

**Detecção:**
```yaml
condition:
  replica_changes: > 5
  time_window: 5 minutos

exemplo:
  t=0:00  → 3 replicas
  t=0:30  → 5 replicas ↑↑
  t=1:00  → 3 replicas ↓↓
  t=2:00  → 6 replicas ↑↑↑
  t=3:00  → 4 replicas ↓↓
  t=4:00  → 7 replicas ↑↑↑

resultado: 🚨 6 mudanças em 5 minutos = THRASHING
```

**Causas Comuns:**
- HPA targets muito sensíveis (ex: 50% CPU)
- Métricas com spikes artificiais (ex: batch jobs)
- `stabilizationWindow` muito curto
- Métricas instáveis (ex: network I/O de API externa)

**Impacto:**
- ⚠️ Pods sendo criados/destruídos constantemente
- 💸 Desperdício de recursos
- 🐛 Bugs por inicialização/finalização frequente
- 📊 Métricas inconsistentes

**Ação Sugerida:**
```yaml
1. Revisar targets do HPA:
   - CPU target < 70%? Considere aumentar para 70-80%
   - Memory target < 80%? Considere aumentar

2. Aumentar stabilizationWindow:
   behavior:
     scaleDown:
       stabilizationWindowSeconds: 300  # 5 minutos

3. Verificar spikes artificiais:
   - Filtrar métricas de batch jobs
   - Usar average em vez de max

4. Considerar custom metrics mais estáveis:
   - Request rate em vez de CPU
   - Queue depth em vez de memory
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  scaling:
    oscillation:
      max_changes: 5           # Máximo de mudanças permitidas
      window_minutes: 5        # Janela de tempo
      severity: critical       # Nível de severidade
      cooldown_minutes: 10     # Não alertar novamente por 10min
```

---

#### B) Maxed Out

**Descrição:** HPA atingiu `maxReplicas` mas métricas continuam altas

**Detecção:**
```yaml
condition:
  current_replicas: == maxReplicas
  AND:
    - cpu_current: > cpu_target + 20%
    OR:
    - memory_current: > memory_target + 20%
  duration: > 2 minutos consecutivos

exemplo:
  maxReplicas: 10
  currentReplicas: 10        ✅ Maxed out
  cpuTarget: 70%
  cpuCurrent: 92%            🚨 22% acima do target!

resultado: 🚨 HPA não pode escalar mais mas carga alta
```

**Causas Comuns:**
- `maxReplicas` muito conservador
- Spike de tráfego além da capacidade planejada
- Problema de performance (não resolve com escala)

**Impacto:**
- 🔥 Aplicação sobrecarregada
- 😡 Usuários com latência alta ou erros
- 💥 Risco de cascata de falhas

**Ação Sugerida:**
```yaml
IMEDIATA:
  1. Aumentar maxReplicas temporariamente:
     kubectl patch hpa <name> -p '{"spec":{"maxReplicas":15}}'

  2. Verificar se há problemas de performance:
     - Queries SQL lentas?
     - Memory leaks?
     - Código ineficiente?

LONGO PRAZO:
  1. Ajustar maxReplicas baseado em carga esperada:
     maxReplicas = peak_load / capacity_per_pod * 1.5

  2. Considerar horizontal + vertical scaling

  3. Otimizar aplicação se problema não é de escala
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  scaling:
    maxed_out:
      cpu_deviation_percent: 20      # Quanto acima do target
      memory_deviation_percent: 20   # Quanto acima do target
      duration_minutes: 2            # Por quanto tempo
      severity: critical
```

---

#### C) Scaling Stuck

**Descrição:** Métricas altas mas HPA não consegue escalar

**Detecção:**
```yaml
condition:
  cpu_current: > cpu_target + 30%
  OR:
  memory_current: > memory_target + 30%

  AND:
  desired_replicas: == current_replicas  # Não mudou
  duration: > 5 minutos

exemplo:
  cpuTarget: 70%
  cpuCurrent: 105%           🚨 35% acima!
  desiredReplicas: 5
  currentReplicas: 5         ⚠️ Não escalou

resultado: 🚨 HPA quer escalar mas não consegue
```

**Causas Comuns:**
- **ResourceQuota excedida** no namespace
- **Insufficient resources** no cluster (nodes cheios)
- **Pod disruption budget** impedindo
- **Taints/tolerations** impedindo scheduling
- HPA `conditions` com erro

**Impacto:**
- 🔥 Aplicação sobrecarregada sem conseguir escalar
- 📉 SLA impactado
- 😡 Usuários insatisfeitos

**Ação Sugerida:**
```yaml
DIAGNÓSTICO:
  1. Verificar events do HPA:
     kubectl describe hpa <name>

  2. Verificar conditions:
     kubectl get hpa <name> -o yaml | grep conditions -A 10

  3. Verificar quotas do namespace:
     kubectl describe quota -n <namespace>

  4. Verificar capacidade do cluster:
     kubectl top nodes
     kubectl describe nodes | grep -A 5 "Allocated resources"

AÇÃO:
  - Quota excedida → Aumentar quota ou limpar recursos
  - Nodes cheios → Adicionar nodes ao cluster
  - PDB restritivo → Ajustar PodDisruptionBudget
  - Taints → Adicionar tolerations aos pods
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  scaling:
    stuck:
      cpu_deviation_percent: 30
      memory_deviation_percent: 30
      duration_minutes: 5
      severity: critical
```

---

### 🟡 WARNINGS

#### D) Underutilization

**Descrição:** Réplicas altas mas métricas muito baixas (desperdício)

**Detecção:**
```yaml
condition:
  cpu_current: < cpu_target - 40%
  OR:
  memory_current: < memory_target - 40%

  AND:
  current_replicas: > minReplicas + 3
  duration: > 15 minutos

exemplo:
  cpuTarget: 70%
  cpuCurrent: 25%            ⚠️ 45% abaixo!
  currentReplicas: 8
  minReplicas: 2

resultado: 🟡 Desperdício de recursos
```

**Causas Comuns:**
- Spike de carga passou mas HPA ainda não scaled down
- `scaleDown.stabilizationWindow` muito longo
- Tráfego sazonal (horário de baixa)
- `maxReplicas` ou `target` mal configurados

**Impacto:**
- 💸 Custo desnecessário (pods ociosos)
- 📊 Métricas enganosas (baixa utilização)

**Ação Sugerida:**
```yaml
SE TEMPORÁRIO (horário baixa):
  - Aguardar HPA scale down naturalmente
  - Considerar scheduled scaling (ex: CronJob)

SE PERSISTENTE:
  1. Reduzir maxReplicas:
     spec:
       maxReplicas: 5  # Era 10

  2. Ou ajustar target:
     spec:
       metrics:
       - type: Resource
         resource:
           name: cpu
           target:
             type: Utilization
             averageUtilization: 80  # Era 70

  3. Ou reduzir stabilizationWindow:
     behavior:
       scaleDown:
         stabilizationWindowSeconds: 60  # Era 300
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  scaling:
    underutilization:
      cpu_deviation_percent: 40
      memory_deviation_percent: 40
      duration_minutes: 15
      min_excess_replicas: 3   # Só alerta se >3 réplicas além do min
      severity: warning
```

---

#### E) Frequent Scaling

**Descrição:** Escala com frequência mas não é thrashing

**Detecção:**
```yaml
condition:
  replica_changes: 3-4
  time_window: 10 minutos

exemplo:
  t=0   → 3 replicas
  t=3m  → 4 replicas ↑
  t=6m  → 5 replicas ↑
  t=9m  → 4 replicas ↓

resultado: 🟡 4 mudanças em 10min (instabilidade leve)
```

**Impacto:**
- 🔄 Churn moderado de pods
- 📊 Métricas um pouco instáveis

**Ação Sugerida:**
```yaml
1. Aumentar stabilizationWindow:
   behavior:
     scaleDown:
       stabilizationWindowSeconds: 180
```

---

## 2️⃣ Anomalias de Deployment/Pods (Application Health)

### 🔴 CRÍTICAS

#### A) Pod CrashLoopBackOff

**Descrição:** Pods reiniciando continuamente após falhas

**Detecção:**
```yaml
condition:
  ANY pod:
    status: CrashLoopBackOff
  OR:
    restart_count: > 5
    time_window: 10 minutos

exemplo:
  pod-1: Running, restarts=0
  pod-2: CrashLoopBackOff, restarts=8  🚨
  pod-3: Running, restarts=1

resultado: 🚨 Pod em crash loop
```

**Causas Comuns:**
- **Application error** - Código crashando
- **Missing dependencies** - DB inacessível, secret faltando
- **Liveness probe failure** - App não responde a tempo
- **Resource limits** - CPU/Memory insuficiente
- **Config error** - Variável de ambiente errada

**Impacto:**
- 📉 Capacidade reduzida (pods não funcionais)
- 🔥 Pode afetar todo deployment se muitos pods crasham
- 😡 Usuários impactados

**Correlação Comum:**
```
🚨 HPA escalou para 10 replicas
⚠️  Mas 6 pods em CrashLoopBackOff
💡 Problema NÃO é carga, é código/configuração!
```

**Ação Sugerida:**
```yaml
DIAGNÓSTICO:
  1. Ver logs do pod:
     kubectl logs <pod-name> --previous

  2. Ver events:
     kubectl describe pod <pod-name>

  3. Verificar liveness probe:
     kubectl get pod <pod-name> -o yaml | grep liveness -A 5

CAUSAS E SOLUÇÕES:
  Application error:
    → Verificar logs e corrigir código

  Missing dependencies:
    → Verificar conectividade: kubectl exec <pod> -- curl <db-url>
    → Verificar secrets: kubectl get secret <name>

  Liveness probe falha:
    → Aumentar initialDelaySeconds
    → Aumentar timeoutSeconds

  Resource limits:
    → Aumentar CPU/Memory limits
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  pods:
    crash_loop:
      max_restarts: 5
      window_minutes: 10
      severity: critical
      immediate: true  # Alerta imediato se status=CrashLoopBackOff
```

---

#### B) Pods OOMKilled

**Descrição:** Pods sendo mortos por falta de memória

**Detecção:**
```yaml
condition:
  ANY pod:
    termination_reason: OOMKilled
  OR:
    memory_usage: > 95% of memory_limit

exemplo:
  pod-1: memory 450Mi / 512Mi (88%) ✅
  pod-2: OOMKilled (was 510Mi / 512Mi) 🚨

resultado: 🚨 Pod killed por OOM
```

**Causas Comuns:**
- **Memory leak** - Aplicação não libera memória
- **Memory limit muito baixo** - App precisa de mais
- **Spike de uso** - Carga pontual alta
- **Large objects** - Cache, buffers grandes

**Impacto:**
- 💥 Pod morto abruptamente (pode corromper dados)
- 📉 Capacidade reduzida
- 🔄 Reinício constante se leak persistente

**Correlação Comum:**
```
🚨 HPA escalou para 10 replicas
⚠️  Mas todos pods OOMKilled após alguns minutos
💡 Memory leak ou limit insuficiente!
```

**Ação Sugerida:**
```yaml
INVESTIGAR:
  1. Verificar histórico de memory usage:
     # Via Prometheus
     container_memory_working_set_bytes{pod="<pod>"}

  2. Verificar se é leak:
     - Memory sobe continuamente?
     - Ou sobe até limit e estabiliza?

SE LIMIT INSUFICIENTE:
  resources:
    limits:
      memory: 1Gi  # Era 512Mi

SE MEMORY LEAK:
  → Investigar código
  → Usar profiler (pprof, heapdump)
  → Corrigir leak

WORKAROUND TEMPORÁRIO:
  → Aumentar memory limit
  → Adicionar restart automático periódico (não ideal!)
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  pods:
    oom_killed:
      immediate: true          # Alerta imediato
      memory_threshold_percent: 95  # Alerta preventivo
      severity: critical
```

---

#### C) Pods Not Ready

**Descrição:** Pods existem mas não estão prontos para receber tráfego

**Detecção:**
```yaml
condition:
  (available_replicas / current_replicas) < 70%
  duration: > 3 minutos

exemplo:
  currentReplicas: 10
  availableReplicas: 4       ⚠️ Apenas 40% pronto!
  readyReplicas: 4

resultado: 🚨 60% dos pods não estão ready
```

**Causas Comuns:**
- **Readiness probe failing** - App não responde
- **Slow startup** - App demora muito pra iniciar
- **Dependencies unavailable** - DB, cache down
- **Resource constraints** - CPU throttling durante startup

**Impacto:**
- 📉 Capacidade real muito menor que esperado
- ⚠️ HPA pode escalar mais mas pods não ficam ready
- 🔥 Sobrecarga nos pods que estão ready

**Correlação Comum:**
```
🚨 HPA escalou para 10 replicas
⚠️  Mas apenas 3 pods ready
💡 Readiness probe ou dependências falhando!
```

**Ação Sugerida:**
```yaml
DIAGNÓSTICO:
  1. Ver conditions dos pods:
     kubectl get pods -o wide
     kubectl describe pod <pod-name>

  2. Ver readiness probe:
     kubectl get pod <pod-name> -o yaml | grep readiness -A 5

  3. Testar probe manualmente:
     kubectl exec <pod> -- curl localhost:8080/health

SOLUÇÕES:
  Readiness probe timeout:
    readinessProbe:
      initialDelaySeconds: 30  # Era 10
      timeoutSeconds: 5        # Era 1
      periodSeconds: 10

  Startup lento:
    startupProbe:  # K8s 1.16+
      initialDelaySeconds: 0
      periodSeconds: 10
      failureThreshold: 30  # 5 minutos total

  Dependências:
    → Verificar conectividade
    → Adicionar init containers se necessário
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  pods:
    not_ready:
      ready_threshold_percent: 70
      duration_minutes: 3
      severity: critical
```

---

### 🟡 WARNINGS

#### D) High Pod Restart Rate

**Descrição:** Pods reiniciando mas não em crash loop total

**Detecção:**
```yaml
condition:
  restart_count: 2-4
  time_window: 15 minutos
  status: NOT CrashLoopBackOff

exemplo:
  pod-1: restarts=2 em 15min
  pod-2: restarts=3 em 15min

resultado: 🟡 Taxa alta de restarts
```

**Causas Comuns:**
- OOM ocasional
- Liveness probe falhas ocasionais
- Deployment rollouts

**Ação Sugerida:**
```yaml
1. Investigar logs para causa
2. Ajustar probes se necessário
3. Monitorar se evolui para crash loop
```

---

#### E) Slow Rollout

**Descrição:** Rolling update demorando muito

**Detecção:**
```yaml
condition:
  updated_replicas: < desired_replicas
  duration: > 10 minutos

exemplo:
  desiredReplicas: 10
  updatedReplicas: 4
  elapsed: 12 minutos

resultado: 🟡 Rollout lento (40% após 12min)
```

**Causas Comuns:**
- `maxUnavailable` muito conservador
- `maxSurge` = 0 (atualiza um por vez)
- Readiness probe com `initialDelaySeconds` alto
- Resources insuficientes para criar novos pods

**Ação Sugerida:**
```yaml
Acelerar rollout:
  strategy:
    rollingUpdate:
      maxSurge: 25%        # Era 0
      maxUnavailable: 25%  # Era 1
```

---

## 3️⃣ Anomalias de Métricas (Performance)

### 🔴 CRÍTICAS

#### A) CPU Throttling Excessivo

**Descrição:** Pods sendo throttled (limitados) pela CPU

**Detecção:**
```yaml
condition:
  cpu_throttling: > 25%
  duration: > 5 minutos

cálculo:
  throttling_percent = (throttled_time / cpu_time) * 100

exemplo:
  cpu_usage: 450m / 500m (90%)    ✅ Uso ok
  cpu_throttling: 35%              🚨 Muito throttled!

resultado: 🚨 Performance degradada por throttling
```

**Causas:**
- CPU `limit` muito baixo
- Bursts de CPU batendo no limit
- CPU limit desnecessário

**Impacto:**
- 📉 Performance ruim mesmo com CPU "disponível"
- ⏱️ Latência alta
- 🐛 Timeouts

**Correlação Comum:**
```
💡 CPU está em 60% mas latência alta
🚨 CPU throttling em 40%!
→ Problema: limit muito baixo, não falta de CPU
```

**Ação Sugerida:**
```yaml
SOLUÇÃO RECOMENDADA:
  resources:
    requests:
      cpu: 500m
    # limits:
    #   cpu: 1000m  ← REMOVER limit de CPU!

  # CPU limits causam throttling desnecessário
  # Melhor: usar apenas requests + QoS Guaranteed

ALTERNATIVA (se limit necessário):
  resources:
    limits:
      cpu: 2000m  # Aumentar significativamente
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  performance:
    cpu_throttling:
      threshold_percent: 25
      duration_minutes: 5
      severity: critical
```

---

#### B) High Error Rate

**Descrição:** Muitos erros HTTP (5xx)

**Detecção:**
```yaml
condition:
  error_rate: > 5%
  duration: > 2 minutos

cálculo:
  error_rate = (5xx_count / total_requests) * 100

exemplo:
  total_requests: 1000 req/s
  5xx_errors: 75 req/s
  error_rate: 7.5%           🚨

resultado: 🚨 Taxa de erro acima do aceitável
```

**Causas Comuns:**
- **Sobrecarga** - Mais carga que capacidade
- **Dependências down** - DB, cache, API externa
- **Bug** - Código com erro
- **Resources** - OOM, throttling

**Impacto:**
- 😡 Usuários recebendo erros
- 📉 SLA impactado
- 💸 Receita perdida

**Correlação Comum:**
```
🚨 Error rate 8% (>5% threshold)
📊 HPA maxed out (10/10 replicas)
💡 Precisa escalar mais OU problema não é escala
```

**Ação Sugerida:**
```yaml
SE HPA MAXED OUT:
  → Aumentar maxReplicas
  → Scale vertical (mais CPU/Memory por pod)

SE NÃO MAXED OUT:
  1. Verificar dependências:
     - DB latency alta?
     - Cache down?
     - API externa com erro?

  2. Verificar logs para causa:
     kubectl logs <pod> | grep "500\|error"

  3. Verificar se é bug recente:
     - Rollout novo?
     - Config change?
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  performance:
    error_rate:
      threshold_percent: 5
      duration_minutes: 2
      severity: critical
```

---

#### C) High Latency (P95)

**Descrição:** Latência P95 muito alta

**Detecção:**
```yaml
condition:
  p95_latency: > threshold_ms
  OR:
  p95_latency: > baseline * 2
  duration: > 3 minutos

exemplo:
  threshold: 1000ms
  baseline: 200ms
  current_p95: 1200ms        🚨 Acima threshold
  current_p95: 450ms         🚨 2.25x baseline

resultado: 🚨 Latência degradada
```

**Causas Comuns:**
- **Sobrecarga** - CPU/Memory high
- **Slow queries** - DB queries ineficientes
- **External API** - Dependência lenta
- **GC pauses** - Garbage collection (Java, Go)
- **Network issues** - Latência de rede

**Impacto:**
- 😡 Experiência do usuário ruim
- ⏱️ Timeouts em cascata
- 📉 Taxa de conversão menor

**Correlação Comum:**
```
🚨 P95 latency 1200ms (threshold 1000ms)
📊 CPU 90% (target 70%)
💡 Escalar pode resolver
```

**Ação Sugerida:**
```yaml
SE CPU/MEMORY HIGH:
  → HPA deve escalar automaticamente
  → Se maxed out, aumentar maxReplicas

SE CPU/MEMORY OK:
  1. Verificar queries lentas:
     # Prometheus query
     histogram_quantile(0.95,
       rate(http_request_duration_seconds_bucket[5m]))

  2. Verificar dependências externas:
     - DB latency?
     - Cache miss rate?
     - External API timeout?

  3. Profile da aplicação:
     - Go: pprof
     - Java: JProfiler
     - Python: cProfile
```

**Threshold Configurável:**
```yaml
anomaly_detection:
  performance:
    latency:
      p95_threshold_ms: 1000
      spike_multiplier: 2.0    # Alerta se >2x baseline
      duration_minutes: 3
      severity: critical

      # Baseline learning (futuro)
      baseline:
        enabled: false
        learn_days: 7
```

---

### 🟡 WARNINGS

#### D) Degrading Performance

**Descrição:** Performance piorando gradualmente

**Detecção:**
```yaml
condition:
  p95_latency: aumentou 20% em 10 minutos
  OR:
  error_rate: aumentou 50% em 10 minutos

exemplo:
  t=0:   p95=200ms, errors=1%
  t=10m: p95=250ms, errors=1.6%

  latency_increase: 25%      🟡
  error_increase: 60%        🟡

resultado: 🟡 Performance degradando
```

**Causas:**
- Tráfego aumentando
- Memory leak gradual
- Cache warming

**Ação Sugerida:**
```yaml
1. Monitorar tendência
2. Preparar para investigação se continuar
3. Verificar se HPA está reagindo
```

---

## 4️⃣ Anomalias de Configuração (Config Issues)

### 🟡 WARNINGS

#### A) Inefficient HPA Config

**Descrição:** Configuração do HPA pode ser otimizada

**Detecção:**
```yaml
condition:
  minReplicas: == maxReplicas
  OR:
  target: < 30% OR > 90%
  OR:
  metrics: apenas CPU (sem memory)

exemplos:
  # HPA inútil
  minReplicas: 5
  maxReplicas: 5              🟡 HPA não faz nada!

  # Target muito baixo
  cpuTarget: 20%              🟡 Desperdício

  # Target muito alto
  cpuTarget: 95%              🟡 Risco de sobrecarga

  # Só CPU (sem memory)
  metrics:
    - cpu: 70%
    # memory: ???            🟡 Deveria ter!
```

**Ação Sugerida:**
```yaml
Min == Max:
  → Remover HPA ou ajustar min/max

Target muito baixo:
  → Aumentar para 70-80%

Target muito alto:
  → Reduzir para 70-80%

Sem memory target:
  spec:
    metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

#### B) Resource Mismatch

**Descrição:** Requests/Limits desbalanceados

**Detecção:**
```yaml
condition:
  cpu_limit / cpu_request: > 4
  OR:
  memory_limit / memory_request: > 2
  OR:
  limits: não definidos

exemplos:
  # CPU limit muito alto
  requests:
    cpu: 100m
  limits:
    cpu: 1000m              🟡 10x! Pode causar throttling

  # Memory limit muito alto
  requests:
    memory: 128Mi
  limits:
    memory: 512Mi           🟡 4x! Risco de OOM no node

  # Sem limits
  requests:
    cpu: 500m
  # limits: ???            🟡 Pode consumir tudo do node
```

**Ação Sugerida:**
```yaml
RECOMENDAÇÃO:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      # CPU: SEM LIMIT (evita throttling)
      memory: 1Gi    # 2x request (proteção OOM)
```

---

#### C) Missing HPA

**Descrição:** Deployment poderia se beneficiar de HPA

**Detecção:**
```yaml
condition:
  deployment_replicas: > 3
  has_hpa: false
  workload_type: Deployment (não StatefulSet)

exemplo:
  apiVersion: apps/v1
  kind: Deployment
  spec:
    replicas: 5              🟡 Fixo, poderia ter HPA!

resultado: 🟡 Considere adicionar HPA
```

**Ação Sugerida:**
```yaml
Criar HPA:
  apiVersion: autoscaling/v2
  kind: HorizontalPodAutoscaler
  metadata:
    name: my-app
  spec:
    scaleTargetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: my-app
    minReplicas: 2
    maxReplicas: 10
    metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

---

## Limites e Thresholds Recomendados

### Configuração Completa (`configs/watchdog.yaml`)

```yaml
anomaly_detection:
  # ==========================================
  # GENERAL SETTINGS
  # ==========================================
  enabled: true
  scan_interval_seconds: 30

  # Cooldown - Evitar spam de alertas
  cooldown:
    same_alert_minutes: 5        # Não alertar mesmo problema <5min
    same_hpa_minutes: 2          # Intervalo mínimo entre alertas do mesmo HPA
    max_alerts_per_hpa: 3        # Máximo alertas ativos por HPA
    max_total_alerts: 100        # Máximo total de alertas ativos

  # ==========================================
  # 1. SCALING ANOMALIES
  # ==========================================
  scaling:
    oscillation:
      enabled: true
      max_changes: 5
      window_minutes: 5
      severity: critical
      cooldown_minutes: 10

    maxed_out:
      enabled: true
      cpu_deviation_percent: 20
      memory_deviation_percent: 20
      duration_minutes: 2
      severity: critical
      cooldown_minutes: 5

    stuck:
      enabled: true
      cpu_deviation_percent: 30
      memory_deviation_percent: 30
      duration_minutes: 5
      severity: critical

    underutilization:
      enabled: true
      cpu_deviation_percent: 40
      memory_deviation_percent: 40
      duration_minutes: 15
      min_excess_replicas: 3
      severity: warning
      cooldown_minutes: 30

    frequent_scaling:
      enabled: true
      max_changes: 3
      window_minutes: 10
      severity: warning
      cooldown_minutes: 15

  # ==========================================
  # 2. POD HEALTH ANOMALIES
  # ==========================================
  pods:
    crash_loop:
      enabled: true
      max_restarts: 5
      window_minutes: 10
      severity: critical
      immediate_on_status: true  # Alerta imediato se status=CrashLoopBackOff

    oom_killed:
      enabled: true
      immediate: true
      memory_warning_percent: 95  # Alerta preventivo
      severity: critical

    not_ready:
      enabled: true
      ready_threshold_percent: 70
      duration_minutes: 3
      severity: critical

    restart_rate:
      enabled: true
      max_restarts: 2
      window_minutes: 15
      severity: warning
      cooldown_minutes: 10

    slow_rollout:
      enabled: true
      max_rollout_minutes: 10
      min_progress_percent: 50  # Pelo menos 50% em 10min
      severity: warning

  # ==========================================
  # 3. PERFORMANCE ANOMALIES
  # ==========================================
  performance:
    cpu_throttling:
      enabled: true
      threshold_percent: 25
      duration_minutes: 5
      severity: critical

    error_rate:
      enabled: true
      threshold_percent: 5
      duration_minutes: 2
      severity: critical
      cooldown_minutes: 5

    latency:
      enabled: true
      p95_threshold_ms: 1000
      p99_threshold_ms: 2000
      spike_multiplier: 2.0      # Alerta se >2x baseline
      duration_minutes: 3
      severity: critical

      # Baseline learning (futuro)
      baseline:
        enabled: false
        learn_days: 7
        percentile: 95

    degrading:
      enabled: true
      latency_increase_percent: 20
      error_increase_percent: 50
      window_minutes: 10
      severity: warning

  # ==========================================
  # 4. CONFIG ANOMALIES
  # ==========================================
  config:
    inefficient_hpa:
      enabled: true
      severity: warning
      checks:
        - min_equals_max
        - target_too_low: 30
        - target_too_high: 90
        - missing_memory_target

    resource_mismatch:
      enabled: true
      severity: warning
      cpu_limit_ratio: 4         # limit/request > 4 = warning
      memory_limit_ratio: 2      # limit/request > 2 = warning

    missing_hpa:
      enabled: true
      severity: info
      min_replicas_to_suggest: 3
      workload_types:
        - Deployment

  # ==========================================
  # ENRICHMENT
  # ==========================================
  enrichment:
    enabled: true
    include_metrics: true
    include_events: true
    include_logs_sample: false   # Futuro: sample de logs
    max_events: 10

  # ==========================================
  # CORRELATION
  # ==========================================
  correlation:
    enabled: true
    group_by:
      - cluster
      - namespace
      - hpa
    correlation_window_minutes: 5
    max_related_alerts: 5
```

---

## Roadmap de Implementação

### **Fase 1: MVP** ✅ (Implementar AGORA)

**Objetivo:** Detectar problemas críticos mais comuns

**Anomalias:**
1. ✅ **Oscillation** - Fácil detectar, muito problemático
2. ✅ **Maxed Out** - Crítico, ação clara
3. ✅ **OOMKilled** - Urgente, fácil detectar
4. ✅ **Pods Not Ready** - Muito comum, importante
5. ✅ **High Error Rate** - Indica problema real

**Arquivos:**
```
internal/analyzer/
├── detector.go       # Engine principal de detecção
├── rules/
│   ├── oscillation.go
│   ├── maxed_out.go
│   ├── oom_killed.go
│   ├── not_ready.go
│   └── error_rate.go
├── models.go         # Anomaly, Alert structs
└── detector_test.go  # Testes unitários
```

**Timeline:** 1-2 dias

---

### **Fase 2: Expansão** (Próxima iteração)

**Objetivo:** Cobrir mais cenários e performance

**Anomalias:**
6. Scaling Stuck
7. CPU Throttling
8. High Latency
9. Underutilization
10. CrashLoopBackOff

**Features:**
- Cooldown de alertas
- Alert deduplication
- Histórico de anomalias

**Timeline:** 2-3 dias

---

### **Fase 3: Features Avançadas** (Futuro)

**Objetivo:** Inteligência e automação

**Features:**

**1. Alert Correlation**
```yaml
# Agrupar alertas relacionados
Em vez de:
  🚨 HPA my-app: maxed out
  🚨 HPA my-app: high CPU
  🚨 HPA my-app: high latency

Criar incident agrupado:
  🚨 INCIDENT #123: my-app capacity issue
     Duration: 5 minutes
     Severity: CRITICAL

     Related alerts:
     ├─ HPA maxed out (10/10 replicas)
     ├─ CPU 92% (target 70%)
     ├─ P95 latency 1200ms (threshold 1000ms)
     └─ Error rate 3.5%

     Root cause analysis:
     💡 HPA cannot scale further (maxReplicas reached)

     Suggested actions:
     1. Increase maxReplicas to 15
     2. Scale vertically (increase CPU per pod)
     3. Investigate if performance issue (not just load)

     Related metrics:
     [ASCII chart of CPU/Latency last 15min]
```

**2. Baseline Learning**
```yaml
# Aprender comportamento normal
baseline:
  enabled: true
  learn_for_days: 7

  patterns:
    - type: daily
      description: "Segunda = mais load que domingo"

    - type: hourly
      description: "9-17h = horário de pico"

    - type: weekly
      description: "Sexta tarde = deploy window"

  # Alertar apenas desvios significativos do normal
  deviation_threshold: 2.0  # 2 desvios padrão

  # Exemplo
  monday_9am:
    cpu_baseline: 75% ± 5%
    current: 78%           ✅ Normal
    current: 92%           🚨 Anomalia (>2σ)
```

**3. Predictive Alerts**
```yaml
# Prever problemas antes de acontecer
prediction:
  enabled: true
  algorithms:
    - linear_regression
    - holt_winters

  predict_minutes: 15
  confidence_threshold: 0.8

  # Exemplo
  prediction:
    type: maxed_out
    confidence: 85%
    eta_minutes: 12

    alert:
      🔮 PREDICTION: my-app will max out in 12 minutes
         Current: 7/10 replicas, CPU 82% (trending up)

         Proactive action:
         kubectl patch hpa my-app -p '{"spec":{"maxReplicas":15}}'
```

**4. Auto-Remediation** (Careful!)
```yaml
# Ações automáticas (com safeguards!)
auto_remediation:
  enabled: false  # Disabled por padrão
  dry_run: true   # Apenas log, não executa

  max_actions_per_hour: 3
  require_confirmation: true

  rules:
    - anomaly: maxed_out
      action: increase_max_replicas
      params:
        increment: 5
        max_allowed: 50

    - anomaly: oom_killed
      action: increase_memory_limit
      params:
        multiplier: 1.5
        max_allowed: 4Gi
```

**5. Slack/Discord/Teams Integration**
```yaml
notifications:
  - type: slack
    webhook: https://hooks.slack.com/...
    channel: "#alerts-prod"
    severity: [critical, warning]

  - type: discord
    webhook: https://discord.com/api/webhooks/...

  - type: teams
    webhook: https://outlook.office.com/webhook/...

# Formato da mensagem
slack_message:
  🚨 *CRITICAL*: my-app maxed out

  *Details:*
  • Cluster: production
  • Namespace: default
  • HPA: my-app
  • Current: 10/10 replicas
  • CPU: 92% (target 70%)

  *Actions:*
  • <kubectl patch ...>
  • <view in grafana>
  • <silence for 1h>
```

**Timeline:** 1-2 semanas

---

## Ideias Futuras

### 1. Machine Learning Anomaly Detection

```yaml
ml_detection:
  enabled: false
  algorithm: isolation_forest

  features:
    - cpu_usage
    - memory_usage
    - request_rate
    - error_rate
    - latency_p95
    - replica_count

  # Treinar modelo com dados históricos
  training:
    days: 30
    retrain_interval_days: 7

  # Detectar anomalias
  anomaly_threshold: 0.8  # Isolation score
```

### 2. Cost Optimization Alerts

```yaml
cost_optimization:
  enabled: true

  checks:
    - type: oversized_pods
      threshold: 50%  # CPU/Memory <50% por 24h

    - type: idle_hpas
      threshold: minReplicas por 7 dias

    - type: expensive_regions
      compare_regions: true

  # Exemplo alert
  💰 COST: my-app is oversized
     Average CPU: 25% (request: 1000m)

     Savings: $450/month by reducing to 500m

     Action:
     kubectl set resources deployment my-app --requests=cpu=500m
```

### 3. Chaos Engineering Integration

```yaml
chaos:
  enabled: false

  experiments:
    - name: pod_failure
      description: "Kill random pods to test resilience"
      validate_hpa_recovers: true

    - name: cpu_stress
      description: "Stress CPU to test HPA scaling"
      validate_scales_up: true

    - name: network_latency
      description: "Add latency to test degradation"
      validate_error_rate_acceptable: true
```

### 4. Multi-Cluster Comparison

```yaml
multi_cluster:
  enabled: true

  compare:
    - cluster: production-us
      with: production-eu

  alerts:
    - type: performance_drift
      threshold: 30%

  # Exemplo
  🌍 DRIFT: my-app performance varies by cluster
     production-us: p95=200ms
     production-eu: p95=450ms  🚨 2.25x slower!

     Investigation needed: network? config? data?
```

### 5. SLO/SLA Tracking

```yaml
slo:
  enabled: true

  objectives:
    - name: api_availability
      target: 99.9%
      window: 30d
      metric: (successful_requests / total_requests)

    - name: api_latency
      target: 95% < 500ms
      window: 7d
      metric: p95_latency

  # Burn rate alerts
  alerts:
    - type: error_budget_burn
      threshold: 10%  # Gastando 10% error budget/hora
```

---

## Referências

### Artigos e Papers
- [Google SRE Book - Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Robust and Effective Logging for Anomaly Detection](https://ieeexplore.ieee.org/document/8418609)
- [HPA Best Practices - Kubernetes Docs](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)

### Ferramentas Similares
- [Goldilocks](https://github.com/FairwindsOps/goldilocks) - VPA recommendations
- [Robusta](https://github.com/robusta-dev/robusta) - Kubernetes troubleshooting
- [Keptn](https://keptn.sh/) - Cloud-native application lifecycle

### Prometheus Queries
- [Awesome Prometheus Alerts](https://awesome-prometheus-alerts.grep.to/)
- [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics)

---

**Última atualização:** 2025-10-25
**Versão:** 1.0
**Status:** 📝 Documentação Completa - Pronto para implementação
