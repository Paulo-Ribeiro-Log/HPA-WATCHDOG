# HPA Watchdog 🐕

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Planning-yellow.svg)]()

**HPA Watchdog** é um monitor autônomo para Horizontal Pod Autoscalers (HPAs) em clusters Kubernetes com interface TUI interativa.

## 🎯 Objetivo

Monitorar continuamente múltiplos clusters Kubernetes, detectando anomalias em HPAs com sistema de alertas centralizado e contexto enriquecido de métricas.

## ✨ Features

- 🔍 **Monitoramento Multi-Cluster**: Monitora múltiplos clusters simultaneamente
- 📊 **Integração Prometheus**: Métricas ricas e análise temporal nativa
- 🚨 **Alertmanager Integration**: Dashboard centralizado de alertas existentes
- 📈 **Análise Temporal**: Histórico de 5 minutos com detecção de tendências
- 🎨 **TUI Interativa**: Interface rica com Bubble Tea + Lipgloss
- 🔔 **Sistema de Alertas**: Detecção complementar de anomalias
- ⚙️ **Configuração Dinâmica**: Ajuste de thresholds via interface
- 🔄 **Auto-Discovery**: Descobre clusters, Prometheus e Alertmanager automaticamente

## 🏗️ Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                        HPA Watchdog                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌───────────────┐  ┌────────────────────┐  │
│  │  K8s API     │  │  Prometheus   │  │  Alertmanager      │  │
│  │  (Config)    │  │  (Metrics)    │  │  (Alerts)          │  │
│  └──────┬───────┘  └───────┬───────┘  └─────────┬──────────┘  │
│         │                   │                     │              │
│         └──────────┬────────┴─────────────────────┘              │
│                    ▼                                             │
│         ┌─────────────────────────┐                             │
│         │   Unified Collector     │                             │
│         └────────────┬────────────┘                             │
│                      ▼                                           │
│         ┌─────────────────────────┐                             │
│         │   Alert Aggregator      │                             │
│         └────────────┬────────────┘                             │
│                      ▼                                           │
│         ┌─────────────────────────┐                             │
│         │   Rich TUI Dashboard    │                             │
│         └─────────────────────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Pré-requisitos

- Go 1.23+
- Acesso a clusters Kubernetes (kubeconfig configurado)
- Prometheus instalado nos clusters (opcional, mas recomendado)
- Alertmanager (opcional)

### Instalação

```bash
# Clone o repositório
git clone https://github.com/Paulo-Ribeiro-Log/hpa-watchdog.git
cd hpa-watchdog

# Baixe dependências
make deps

# Build
make build

# Ou instale no GOPATH
make install
```

### Uso Básico

```bash
# Rodar com configuração padrão
./build/hpa-watchdog

# Com configuração customizada
./build/hpa-watchdog --config /path/to/config.yaml

# Debug mode
./build/hpa-watchdog --debug

# Validar configuração
make validate
```

## ⚙️ Configuração

Arquivo de exemplo: `configs/watchdog.yaml`

```yaml
monitoring:
  scan_interval_seconds: 30
  history_retention_minutes: 5

  prometheus:
    enabled: true
    auto_discover: true
    fallback_to_metrics_server: true

  alertmanager:
    enabled: true
    auto_discover: true
    sync_interval_seconds: 30

thresholds:
  cpu_warning_percent: 85
  cpu_critical_percent: 90
  replica_delta_percent: 50.0
```

## 🎨 Interface TUI

### Views Principais

1. **Dashboard**: Overview de todos os clusters com alertas recentes
2. **Alerts**: Lista detalhada de alertas ativos (Alertmanager + Watchdog)
3. **Clusters**: Breakdown por cluster e namespace
4. **Config**: Modal interativo para ajustar thresholds

### Controles

| Tecla | Ação |
|-------|------|
| `Tab` | Alternar views |
| `↑↓` | Navegar |
| `Enter` | Ver detalhes |
| `A` | Acknowledge alerta |
| `S` | Silenciar (Alertmanager) |
| `E` | Enriquecer com contexto |
| `Q` | Sair |

## 📊 Métricas Monitoradas

### Kubernetes API
- HPA config (min/max replicas, targets)
- Current/Desired replicas
- Deployment resources (requests/limits)
- Events

### Prometheus
- CPU/Memory usage (atual + histórico 5min)
- Request rate (QPS)
- Error rate (5xx)
- P95 Latency
- Network I/O

### Alertmanager
- Alertas existentes de regras Prometheus
- Status de silenciamentos
- Histórico de alertas

## 🔍 Detecção de Anomalias

### Alertmanager (Fonte Primária - 70%)
Sincroniza e enriquece alertas existentes das regras Prometheus

### Watchdog (Complementar - 30%)
- Replica Oscillation (mudanças rápidas)
- Scaling Stuck (HPA não consegue escalar)
- Target Deviation (desvio do target)
- Config Changes (mudanças em HPA/deployment)
- Complex Correlations (múltiplos indicadores)

## 🛠️ Development

```bash
# Rodar testes
make test

# Testes curtos (sem integração)
make test-short

# Coverage
make coverage

# Lint
make lint

# Format
make fmt
```

## 📦 Build

```bash
# Build local
make build

# Build para Linux
make build-linux

# Build para macOS
make build-darwin

# Build para todas plataformas
make build-all
```

## 🐳 Docker

```bash
# Build imagem
make docker-build

# Rodar container
make docker-run
```

## 🔐 Permissões Kubernetes

O Watchdog requer apenas permissões de **leitura**:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hpa-watchdog-reader
rules:
- apiGroups: [""]
  resources: ["namespaces", "pods"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets"]
  verbs: ["get", "list"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
```

## 📖 Documentação

- [CLAUDE.md](CLAUDE.md) - Guia para desenvolvimento com Claude Code
- [HPA_WATCHDOG_SPEC.md](HPA_WATCHDOG_SPEC.md) - Especificação técnica completa
- [HPA_WATCHDOG_PROMETHEUS_ANALYSIS.md](HPA_WATCHDOG_PROMETHEUS_ANALYSIS.md) - Análise de integração Prometheus
- [HPA_WATCHDOG_ALERTMANAGER.md](HPA_WATCHDOG_ALERTMANAGER.md) - Integração Alertmanager

## 🎯 Filosofia: KISS

Este projeto segue o princípio **Keep It Simple, Stupid**:
- Código simples e direto sobre soluções "inteligentes"
- Sem over-engineering
- Tecnologia confiável e testada
- Fail fast com mensagens claras

## 🗺️ Roadmap

### Fase 1: MVP (Em Desenvolvimento)
- [x] Setup projeto
- [ ] Core monitoring (K8s + Prometheus)
- [ ] Alertmanager client
- [ ] TUI básico
- [ ] Config system

### Fase 2: Features Avançadas
- [ ] Silence management
- [ ] Alert correlation
- [ ] Enhanced UI com ASCII charts
- [ ] SQLite persistence

### Fase 3: Production Ready
- [ ] Systemd service
- [ ] Docker image
- [ ] Webhook notifications
- [ ] Performance optimization

## 📄 Licença

MIT License - Veja [LICENSE](LICENSE) para detalhes

## 🤝 Contribuindo

Contribuições são bem-vindas! Por favor:

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -m 'feat: adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 👤 Autor

**Paulo Ribeiro**

- GitHub: [@Paulo-Ribeiro-Log](https://github.com/Paulo-Ribeiro-Log)

## 🙏 Agradecimentos

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Framework TUI
- [Kubernetes client-go](https://github.com/kubernetes/client-go) - K8s API
- [Prometheus](https://prometheus.io/) - Metrics & Monitoring

---

**Status**: 🟡 Em desenvolvimento ativo
# HPA-WATCHDOG
