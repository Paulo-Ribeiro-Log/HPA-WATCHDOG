# HPA Watchdog - Documentação

Índice completo da documentação do projeto.

## 📚 Documentação Principal

### Começando

- **[../README.md](../README.md)** - Overview do projeto, instalação e quick start
- **[../CLAUDE.md](../CLAUDE.md)** - Guia completo para desenvolvimento (arquitetura, padrões, etc)
- **[../PROJECT_STATUS.md](../PROJECT_STATUS.md)** - Status atual do projeto

### Desenvolvimento

- **[TESTING.md](./TESTING.md)** - Guia completo de testes
  - Comando `test` focado
  - Variáveis de ambiente
  - Troubleshooting
  - Integração CI/CD

### Detecção de Anomalias

- **[ANOMALY_DETECTION.md](./ANOMALY_DETECTION.md)** - ⭐ Documentação COMPLETA
  - Filosofia (Signal vs Noise)
  - 4 Categorias de anomalias
  - Detecção, limites e ações
  - Roadmap de implementação
  - Ideias futuras (ML, auto-remediation, etc)

- **[ANOMALY_DETECTION_SUMMARY.md](./ANOMALY_DETECTION_SUMMARY.md)** - Referência rápida
  - Quick reference table
  - Decision matrix
  - Alert priority
  - Common correlations

## 📁 Estrutura de Documentação

```
docs/
├── README.md                        # Este arquivo (índice)
├── TESTING.md                       # Guia de testes
├── ANOMALY_DETECTION.md             # Spec completa de anomalias ⭐
└── ANOMALY_DETECTION_SUMMARY.md     # Cheat sheet

HPA-Watchdog/
├── README.md                        # Overview do projeto
├── CLAUDE.md                        # Guia desenvolvimento ⭐
├── PROJECT_STATUS.md                # Status atual
├── CONTRIBUTING.md                  # Como contribuir
├── HPA_WATCHDOG_*.md                # Specs técnicas originais
└── configs/
    └── watchdog.yaml                # Configuração de exemplo
```

## 🎯 Por Onde Começar?

### Se você é novo no projeto:

1. **[README.md](../README.md)** - Entenda o que é o HPA Watchdog
2. **[CLAUDE.md](../CLAUDE.md)** - Arquitetura e filosofia KISS
3. **[PROJECT_STATUS.md](../PROJECT_STATUS.md)** - O que já está pronto

### Se você vai implementar anomalias:

1. **[ANOMALY_DETECTION.md](./ANOMALY_DETECTION.md)** - Leia TUDO ⭐
2. **[ANOMALY_DETECTION_SUMMARY.md](./ANOMALY_DETECTION_SUMMARY.md)** - Keep como referência
3. **[CLAUDE.md](../CLAUDE.md)** - Seção "Common Patterns"

### Se você vai testar:

1. **[TESTING.md](./TESTING.md)** - Guia completo de testes
2. **[CLAUDE.md](../CLAUDE.md)** - Seção "Development Commands"

### Se você vai contribuir:

1. **[CONTRIBUTING.md](../CONTRIBUTING.md)** - Guidelines
2. **[CLAUDE.md](../CLAUDE.md)** - Padrões de código

## 📖 Specs Originais

Documentos de especificação técnica (referência):

- **[HPA_WATCHDOG_SPEC.md](../HPA_WATCHDOG_SPEC.md)** - Especificação técnica completa original
- **[HPA_WATCHDOG_PROMETHEUS_ANALYSIS.md](../HPA_WATCHDOG_PROMETHEUS_ANALYSIS.md)** - Análise integração Prometheus
- **[HPA_WATCHDOG_ALERTMANAGER.md](../HPA_WATCHDOG_ALERTMANAGER.md)** - Integração Alertmanager

## 🔧 READMEs de Packages

Documentação de cada package interno:

- **[../internal/monitor/README.md](../internal/monitor/README.md)** - K8s Client + Port-Forward Manager
- **[../internal/prometheus/README.md](../internal/prometheus/README.md)** - Prometheus Client + Queries

## 💡 Decisões de Design

### Por que HPA e não Deployment?
- **Decisão:** Monitorar HPAs primariamente, deployments secundariamente
- **Razão:** Foco no comportamento de autoscaling
- **Docs:** [ANOMALY_DETECTION.md](./ANOMALY_DETECTION.md) - Introdução

### Por que Port-Forward na porta 55553?
- **Decisão:** Porta fixa 55553 com heartbeat
- **Razão:** Evita conflitos + lifecycle management
- **Docs:** [../internal/monitor/README.md](../internal/monitor/README.md)

### Por que KISS (Keep It Simple)?
- **Decisão:** Simplicidade sobre cleverness
- **Razão:** Manutenibilidade, confiabilidade
- **Docs:** [../CLAUDE.md](../CLAUDE.md) - Core Philosophy

### Por que Prometheus + Alertmanager?
- **Decisão:** Híbrido - 70% Alertmanager + 30% Watchdog detection
- **Razão:** Aproveitar alertas existentes + detectar padrões complexos
- **Docs:** [../CLAUDE.md](../CLAUDE.md) - Architecture

## 🗺️ Roadmap

### ✅ Concluído

- [x] Estrutura base do projeto
- [x] Config loader (Viper)
- [x] Cluster discovery
- [x] K8s Client
- [x] Port-Forward Manager com heartbeat
- [x] Prometheus Client
- [x] Auto-discovery Prometheus
- [x] 17 PromQL queries predefinidas
- [x] Comando `test` focado
- [x] Documentação completa de anomalias

### 🚧 Em Desenvolvimento

- [ ] Analyzer de anomalias (Fase 1 - 5 anomalias MVP)
- [ ] TUI básico (Bubble Tea)
- [ ] Alertmanager client

### 📋 Próximos Passos

**Fase 1 - MVP:**
1. Implementar Analyzer (5 anomalias críticas)
2. TUI Dashboard básico
3. Alertmanager integration
4. Testes em cluster real

**Fase 2 - Advanced:**
5. 5 anomalias adicionais
6. Alert correlation
7. TUI avançado (charts, history)
8. Persistence (SQLite)

**Fase 3 - Production:**
9. Systemd service
10. Docker image
11. Webhook notifications
12. Performance optimization

Detalhes: [PROJECT_STATUS.md](../PROJECT_STATUS.md)

## 🧪 Testing

### Unit Tests

```bash
# Todos os testes
go test ./... -v

# Package específico
go test ./internal/prometheus/... -v

# Com coverage
go test ./... -cover
```

### Integration Tests

```bash
# Teste focado em cluster/namespace
./build/hpa-watchdog test \
  --cluster minikube \
  --namespace default \
  --prometheus \
  --history
```

Mais em: [TESTING.md](./TESTING.md)

## 📊 Métricas e Queries

### Métricas Coletadas

**Do Kubernetes:**
- HPA config, status, replicas
- Deployment resources
- Pod health

**Do Prometheus:**
- CPU/Memory usage (atual + 5min history)
- Request rate, Error rate, P95 latency
- Network I/O
- Pod restarts, OOM events

### PromQL Queries

17 queries predefinidas documentadas em:
- [../internal/prometheus/README.md](../internal/prometheus/README.md)

## 🎨 TUI (Futuro)

### Views Planejadas

1. **Dashboard** - Overview multi-cluster
2. **Alerts** - Lista de alertas ativos
3. **Clusters** - Breakdown por cluster
4. **Config** - Ajuste de thresholds

### Keyboard Controls

```
Tab      - Switch views
↑↓ / jk  - Navigate
Enter    - Details
A        - Acknowledge
S        - Silence
Q        - Quit
```

Mais em: [../CLAUDE.md](../CLAUDE.md) - TUI Navigation

## 🔐 Segurança e Permissões

### RBAC Necessário

```yaml
- namespaces, pods: get, list
- deployments, replicasets: get, list
- horizontalpodautoscalers: get, list
- metrics (metrics.k8s.io): get, list
```

**Nota:** Apenas read-only, sem write permissions!

Mais em: [../CLAUDE.md](../CLAUDE.md) - Security & Permissions

## 🐛 Troubleshooting

### Problemas Comuns

**Cluster não conecta:**
- Verificar kubeconfig: `kubectl config get-contexts`
- Testar acesso: `kubectl cluster-info`

**Prometheus não encontrado:**
- Verificar pod: `kubectl get pods -n monitoring | grep prometheus`
- Verificar service: `kubectl get svc -n monitoring`
- Testar port-forward manual

**Port 55553 em uso:**
- Ver processo: `lsof -i :55553`
- Matar: `kill <PID>`
- Ou usar porta diferente: `export LOCAL_PORT=55554`

Mais em: [TESTING.md](./TESTING.md) - Troubleshooting

## 🤝 Contribuindo

Veja [CONTRIBUTING.md](../CONTRIBUTING.md) para:
- Como contribuir
- Code style
- Pull request process
- Commit message format

## 📝 Changelog

Veja [PROJECT_STATUS.md](../PROJECT_STATUS.md) para:
- Versões
- Features adicionadas
- Bugs corrigidos

## 📫 Contato

- GitHub Issues: [HPA-Watchdog Issues](https://github.com/Paulo-Ribeiro-Log/hpa-watchdog/issues)
- Autor: Paulo Ribeiro

---

## 🌟 Documentos Principais (Must Read)

Para ter visão completa do projeto, leia na ordem:

1. **[../README.md](../README.md)** - O que é e como usar
2. **[../CLAUDE.md](../CLAUDE.md)** - Arquitetura e padrões
3. **[ANOMALY_DETECTION.md](./ANOMALY_DETECTION.md)** - ⭐ Detecção completa
4. **[TESTING.md](./TESTING.md)** - Como testar
5. **[PROJECT_STATUS.md](../PROJECT_STATUS.md)** - Status atual

---

**Última atualização:** 2025-10-25
**Versão da doc:** 1.0
**Status:** 📚 Documentação Completa
