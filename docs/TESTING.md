# Guia de Testes - HPA Watchdog

Este documento explica como testar o HPA Watchdog em um cluster/namespace/deployment específico.

## Comando `test`

O comando `test` permite testar a conectividade e coleta de métricas de forma focada em um HPA específico.

### Uso Básico

```bash
# Testar todos HPAs de um namespace
./build/hpa-watchdog test --cluster production --namespace default

# Testar HPA específico
./build/hpa-watchdog test --cluster production --namespace default --hpa my-app

# Com métricas do Prometheus
./build/hpa-watchdog test --cluster production --namespace default --prometheus

# Mostrar histórico de 5 minutos
./build/hpa-watchdog test --cluster production --namespace default --history
```

### Flags Disponíveis

| Flag | Abreviação | Descrição | Obrigatório |
|------|------------|-----------|-------------|
| `--cluster` | `-c` | Context do kubeconfig | ✅ Sim |
| `--namespace` | `-n` | Namespace do HPA | ✅ Sim |
| `--hpa` | `-H` | Nome do HPA (testa todos se vazio) | ❌ Não |
| `--prometheus` | `-p` | Coletar métricas do Prometheus | ❌ Não |
| `--history` | (sem abr.) | Mostrar histórico de 5 minutos | ❌ Não |
| `--verbose` | `-v` | Logs verbosos | ❌ Não |
| `--debug` | | Debug mode (global) | ❌ Não |

### Variáveis de Ambiente

Alternativamente, você pode configurar via variáveis de ambiente:

```bash
# Configurar target
export TEST_CLUSTER_CONTEXT=production
export TEST_NAMESPACE=default
export TEST_HPA_NAME=my-app

# Prometheus
export PROMETHEUS_NAMESPACE=monitoring
export PROMETHEUS_SERVICE=prometheus-server
export USE_PORT_FORWARD=true
export LOCAL_PORT=55553

# Comportamento
export COLLECT_METRICS=true
export SHOW_HISTORY=true
export VERBOSE=true

# Executar
./build/hpa-watchdog test
```

## Exemplos Práticos

### 1. Teste Básico (Só K8s API)

Testa conectividade e coleta dados básicos do HPA:

```bash
./build/hpa-watchdog test \
  --cluster akspriv-faturamento-prd-admin \
  --namespace ingress-nginx \
  --hpa nginx-ingress-controller
```

**Output (com logs filtrados):**
```
🧪 HPA Watchdog - Teste Integrado
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Cluster:    akspriv-faturamento-prd-admin
   Namespace:  ingress-nginx
   HPA:        nginx-ingress-controller
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Cluster conectado

✅ 1 HPA(s) encontrado(s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 HPA 1/1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Nome: ingress-nginx/nginx-ingress-controller
🕐 Timestamp: 2025-10-24 23:12:37

⚙️  Configuração:
   Min/Max Replicas:  3 / 20
   CPU Target:        60%

📊 Status Atual:
   Current Replicas:  3
   Desired Replicas:  3
   Ready:             true
   Scaling Active:    true
   Last Scale:        2025-03-20 00:10:46 (7mo ago)

💾 Resources (por pod):
   CPU Request:       384m
   CPU Limit:         512m
   Memory Request:    256Mi
   Memory Limit:      384Mi

🔍 Análise Rápida:
   🟡 CONFIG: Memory target não configurado

✅ Teste concluído com sucesso!
```

**Dica:** Para remover os logs do stderr e ver apenas a saída limpa:
```bash
./build/hpa-watchdog test --cluster akspriv-faturamento-prd-admin --namespace ingress-nginx --hpa nginx-ingress-controller 2>/dev/null
```

### 2. Teste com Prometheus

Coleta métricas completas via Prometheus:

```bash
./build/hpa-watchdog test \
  --cluster my-cluster \
  --namespace production \
  --hpa my-app \
  --prometheus
```

**Output esperado:**
```
🧪 Testing HPA Monitoring
   Cluster:    my-cluster
   Namespace:  production
   HPA:        my-app
   Prometheus: enabled

{"level":"info","time":"...","message":"Setting up Prometheus..."}
{"level":"info","local_port":55553,"time":"...","message":"PortForward manager initialized"}
{"level":"info","cluster":"my-cluster","namespace":"monitoring","service":"prometheus-server","local_port":55553,"remote_port":9090,"time":"...","message":"Port-forward started"}
{"level":"info","endpoint":"http://localhost:55553","time":"...","message":"✅ Prometheus connection OK"}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 HPA: production/my-app
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔢 Replicas:
   Min/Max:        2 / 10
   Current:        3
   Desired:        3

📈 Current Metrics (Prometheus):
   CPU:            68.50%
   Memory:         72.30%

🌐 Application Metrics:
   Request Rate:   150.25 req/s
   Error Rate:     0.15%
   P95 Latency:    45.20 ms

✅ Status:
   Ready:          true
   Scaling Active: true
```

### 3. Teste com Histórico

Mostra histórico de 5 minutos:

```bash
./build/hpa-watchdog test \
  --cluster my-cluster \
  --namespace production \
  --hpa my-app \
  --prometheus \
  --history
```

**Output adicional:**
```
📊 CPU History (last 5min):
   T-300s: 65.20%
   T-270s: 66.80%
   T-240s: 68.50%
   T-210s: 70.10%
   T-180s: 71.20%
   T-150s: 69.50%
   T-120s: 68.30%
   T-90s:  67.80%
   T-60s:  68.10%
   T-30s:  68.50%

📊 Replica History (last 5min):
   T-300s: 3
   T-270s: 3
   T-240s: 3
   T-210s: 3
   T-180s: 3
   T-150s: 3
   T-120s: 3
   T-90s:  3
   T-60s:  3
   T-30s:  3
```

### 4. Teste Todos HPAs de um Namespace

```bash
./build/hpa-watchdog test \
  --cluster my-cluster \
  --namespace production \
  --prometheus
```

Isso testará todos os HPAs encontrados no namespace `production`.

## Configuração de Clusters

### Verificar clusters disponíveis

```bash
./build/hpa-watchdog clusters
```

Output:
```
📊 Found 3 cluster(s):

1. production (default)
   Context:   production-context
   Server:    https://k8s-prod.example.com

2. staging
   Context:   staging-context
   Server:    https://k8s-stg.example.com

3. development
   Context:   dev-context
   Server:    https://k8s-dev.example.com
```

## Troubleshooting

### Erro: cluster context não encontrado

```
❌ Failed to create K8s client: failed to create client config for cluster production: context "production" does not exist
```

**Solução:**
```bash
# Verificar contexts disponíveis
kubectl config get-contexts

# Usar o context correto
./build/hpa-watchdog test --cluster <context-name> --namespace default
```

### Erro: namespace não existe

```
❌ Failed to list HPAs: namespaces "production" not found
```

**Solução:**
```bash
# Verificar namespaces disponíveis
kubectl get namespaces

# Usar namespace correto
./build/hpa-watchdog test --cluster my-cluster --namespace <existing-namespace>
```

### Erro: HPA não encontrado

```
❌ HPA my-app not found in namespace production
```

**Solução:**
```bash
# Listar HPAs disponíveis
kubectl get hpa -n production

# Usar nome correto ou omitir --hpa para testar todos
./build/hpa-watchdog test --cluster my-cluster --namespace production
```

### Prometheus não disponível

```
⚠️  Failed to setup port-forward, skipping Prometheus
```

**Soluções:**

1. Verificar se Prometheus está rodando:
```bash
kubectl get pods -n monitoring | grep prometheus
```

2. Verificar serviço:
```bash
kubectl get svc -n monitoring | grep prometheus
```

3. Configurar manualmente:
```bash
export PROMETHEUS_NAMESPACE=kube-prometheus
export PROMETHEUS_SERVICE=kube-prometheus-stack-prometheus
./build/hpa-watchdog test --cluster my-cluster --namespace production --prometheus
```

4. Testar port-forward manual:
```bash
kubectl port-forward -n monitoring svc/prometheus-server 9090:9090
curl http://localhost:9090/api/v1/query?query=up
```

### Port 55553 em uso

```
❌ Failed to start port-forward: bind: address already in use
```

**Solução:**
```bash
# Verificar o que está usando a porta
lsof -i :55553

# Matar processo
kill <PID>

# Ou usar porta diferente
export LOCAL_PORT=55554
./build/hpa-watchdog test --cluster my-cluster --namespace production --prometheus
```

## Scripts de Teste Rápido

### test-local.sh

Crie um script para testes rápidos:

```bash
#!/bin/bash
# test-local.sh

export TEST_CLUSTER_CONTEXT=minikube
export TEST_NAMESPACE=default
export COLLECT_METRICS=true
export SHOW_HISTORY=true
export VERBOSE=true

./build/hpa-watchdog test
```

### test-production.sh

```bash
#!/bin/bash
# test-production.sh

./build/hpa-watchdog test \
  --cluster production \
  --namespace production \
  --hpa api-gateway \
  --prometheus \
  --history \
  --verbose
```

## Integração com CI/CD

### GitHub Actions

```yaml
name: Test HPA Monitoring
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Setup kubectl
        uses: azure/setup-kubectl@v1

      - name: Configure kubeconfig
        run: |
          echo "${{ secrets.KUBECONFIG }}" > kubeconfig
          export KUBECONFIG=./kubeconfig

      - name: Build
        run: make build

      - name: Test HPA Monitoring
        run: |
          ./build/hpa-watchdog test \
            --cluster staging \
            --namespace default \
            --prometheus
```

## Métricas Coletadas

O comando `test` coleta e exibe as seguintes métricas:

### Do Kubernetes API:
- Min/Max replicas
- Current/Desired replicas
- CPU/Memory targets
- Resource requests/limits
- Status (Ready, Scaling Active)
- Last scale time

### Do Prometheus (com --prometheus):
- CPU usage atual (%)
- Memory usage atual (%)
- CPU history (5min, 10 pontos)
- Memory history (5min, 10 pontos)
- Replica history (5min, 10 pontos)
- Request rate (req/s)
- Error rate (%)
- P95 latency (ms)

## Análise Rápida de Anomalias

O comando `test` inclui análise automática de anomalias comuns. Veja exemplos de detecção:

### Anomalias Detectadas

#### 1. Maxed Out (🔴 Crítico)
HPA no limite máximo de réplicas com CPU acima do target:
```
🔍 Análise Rápida:
   🔴 MAXED OUT: no limite (20) com CPU 85.23% (target: 60%)
```
**Ação:** Aumentar `maxReplicas` ou verificar capacidade do cluster

#### 2. Underutilization (🟡 Warning)
CPU muito abaixo do target com muitas réplicas:
```
🔍 Análise Rápida:
   🟡 UNDERUTILIZED: CPU 15.30% muito abaixo do target 60%
```
**Ação:** Reduzir `minReplicas` ou ajustar target

#### 3. Missing Memory Target (🟡 Config)
HPA sem memory target configurado:
```
🔍 Análise Rápida:
   🟡 CONFIG: Memory target não configurado
```
**Ação:** Adicionar `memory` target ao HPA para autoscaling mais preciso

#### 4. High Error Rate (🔴 Crítico)
Taxa de erros 5xx acima de 5%:
```
🔍 Análise Rápida:
   🔴 HIGH ERROR RATE: 8.45% (crítico >5%)
```
**Ação:** Investigar causa dos erros, considerar scale up

#### 5. High Latency (🔴 Crítico)
P95 latency acima de 1000ms:
```
🔍 Análise Rápida:
   🔴 HIGH LATENCY: P95 1523.45ms (>1000ms)
```
**Ação:** Scale up ou investigar gargalos da aplicação

#### 6. Oscillation (🔴 Crítico)
Réplicas mudando frequentemente (>3 vezes em 5min):
```
🔍 Análise Rápida:
   🔴 OSCILLATION: 5 mudanças de réplicas em 5min
```
**Ação:** Aumentar `stabilizationWindow` do HPA

### Exemplo de Saída Completa

```bash
./build/hpa-watchdog test --cluster production --namespace api --hpa gateway --prometheus --history 2>/dev/null
```

**Saída:**
```
📊 HPA 1/1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Nome: api/gateway
🕐 Timestamp: 2025-10-24 23:30:45

⚙️  Configuração:
   Min/Max Replicas:  2 / 20
   CPU Target:        70%
   Memory Target:     80%

📊 Status Atual:
   Current Replicas:  20
   Desired Replicas:  20
   Ready:             true
   Scaling Active:    true
   Last Scale:        2025-10-24 23:28:15 (2m ago)

💾 Resources (por pod):
   CPU Request:       500m
   CPU Limit:         1000m
   Memory Request:    512Mi
   Memory Limit:      1Gi

📈 Métricas (Prometheus):
   CPU Atual:         92.45% (target: 70%, desvio: +22.45%)
   Memory Atual:      75.20% (target: 80%, desvio: -4.80%)

🌐 Métricas Estendidas:
   Request Rate:      1524.30 req/s
   Error Rate:        8.75%
   P95 Latency:       1845.23 ms

📊 Histórico CPU (5 min):
   T-300s: 68.23%
   T-270s: 72.45%
   T-240s: 78.90%
   T-210s: 85.12%
   T-180s: 88.56%
   T-150s: 90.23%
   T-120s: 91.78%
   T-90s: 92.10%
   T-60s: 92.35%
   T-30s: 92.45%

📊 Histórico Replicas (5 min):
   T-300s: 12
   T-270s: 14
   T-240s: 16
   T-210s: 18
   T-180s: 18
   T-150s: 19
   T-120s: 20
   T-90s: 20
   T-60s: 20
   T-30s: 20

🔍 Análise Rápida:
   🔴 MAXED OUT: no limite (20) com CPU 92.45% (target: 70%)
   🔴 HIGH ERROR RATE: 8.75% (crítico >5%)
   🔴 HIGH LATENCY: P95 1845.23ms (>1000ms)
```

**Interpretação:**
- HPA está no limite e não consegue escalar mais
- Alta carga (CPU 92%, crescendo consistentemente)
- Degradação de performance (erros 8.75%, latência >1.8s)
- **Ação urgente:** Aumentar `maxReplicas` ou adicionar capacidade ao cluster

## Próximos Passos

Após validar que o teste funciona:

1. Execute o watchdog completo:
```bash
./build/hpa-watchdog
```

2. Configure thresholds personalizados em `configs/watchdog.yaml`

3. Configure alertas customizados

4. Integre com Alertmanager para alertas centralizados
