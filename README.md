# HPA Watchdog 🐕

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Development-green.svg)]()

**HPA Watchdog** é um monitor autônomo para Horizontal Pod Autoscalers (HPAs) em clusters Kubernetes com interface TUI interativa rica e modo de stress test integrado.

## 🎯 Objetivo

Monitorar continuamente múltiplos clusters Kubernetes, detectando anomalias em HPAs com sistema de alertas centralizado e contexto enriquecido de métricas.

## ✨ Features

- 🔍 **Monitoramento Multi-Cluster**: Monitora múltiplos clusters simultaneamente
- 📊 **Integração Prometheus**: Métricas ricas e análise temporal nativa com port-forward automático
- 🚨 **Alertmanager Integration**: Dashboard centralizado de alertas existentes
- 📈 **Análise Temporal**: Gráficos de séries temporais com timezone GMT-3 (CPU, Memory, Réplicas)
- 🎨 **TUI Interativa**: 7 views implementadas com Bubble Tea + Lipgloss + ntcharts
- 🔔 **Sistema de Alertas**: Detecção de 10 tipos de anomalias (persistentes + mudanças súbitas)
- 💾 **Persistência SQLite**: Auto-save/load com retenção de 24h e limpeza automática
- 🧪 **Modo Stress Test**: Baseline capture, monitoramento em tempo real, relatório automático PASS/FAIL
- 🔄 **Restart Automático**: Reinício inteligente de testes com Shift+R
- 📊 **Scroll Inteligente**: Menu de HPAs com viewport e indicadores visuais
- 📋 **Relatórios Detalhados**: Resumo executivo, métricas de pico PRE→PEAK→POST, recomendações

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

### Views Implementadas (7 views)

1. **Setup**: Configuração inicial interativa (clusters, modo, duração, intervalo)
2. **Dashboard**: Overview multi-cluster com top clusters, anomalias recentes
3. **Alertas**: Lista detalhada com filtragem por severidade/cluster
4. **Clusters**: Breakdown por cluster e namespace com métricas agregadas
5. **Histórico**: Gráficos temporais de CPU/Memory/Réplicas (timezone GMT-3)
6. **Stress Test**: Dashboard em tempo real com baseline e gráficos interativos
7. **Relatório Final**: Resumo executivo PASS/FAIL com métricas PRE→PEAK→POST

### Controles

#### Gerais
| Tecla | Ação |
|-------|------|
| `Tab` | Alternar views |
| `↑↓` ou `j k` | Navegar (scroll automático) |
| `Enter` | Selecionar/Detalhar |
| `H` / `Home` | Volta ao Dashboard |
| `F5` / `R` | Refresh |
| `Q` / `Ctrl+C` | Sair |

#### Alertas
| Tecla | Ação |
|-------|------|
| `A` | Acknowledge alerta |
| `Shift+A` | Acknowledge todos |
| `S` | Silenciar (Alertmanager) |
| `C` | Limpar reconhecidos |
| `E` | Enriquecer com contexto |

#### Stress Test
| Tecla | Ação |
|-------|------|
| `P` | Pausar/Retomar scan |
| `Shift+R` | Reiniciar teste |
| `E` | Exportar Markdown (TODO) |
| `Shift+E` | Exportar PDF (TODO) |

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

### Watchdog Analyzer (Complementar - 30%)

#### Fase 1: Anomalias de Estado Persistente
1. **Oscilação**: >5 alterações de réplica em 5min
2. **No Limite**: Réplicas = máx + CPU > alvo +20% por 2min
3. **OOMKilled**: Pod finalizado por falta de memória
4. **Pods Não Prontos**: Pods não prontos por 3min+
5. **Alta Taxa de Erros**: >5% de erros 5xx por 2min

#### Fase 2: Mudanças Súbitas (scan a scan)
6. **Pico de CPU**: CPU aumentou >50% entre scans
7. **Pico de Réplicas**: Réplicas aumentaram +3 entre scans
8. **Pico de Erros**: Taxa de erros aumentou >5% entre scans
9. **Pico de Latência**: Latência aumentou >100% entre scans
10. **Queda de CPU**: CPU caiu >50% entre scans

**Total**: 10 tipos de anomalia implementados e testados

## 🧪 Modo Stress Test

O HPA Watchdog inclui um modo especializado para testes de carga e validação de configurações de HPA.

### Funcionalidades

1. **Baseline Capture Automático**: Captura estado PRE (réplicas, CPU, memory) dos últimos 30min
2. **Monitoramento em Tempo Real**: Dashboard interativo com gráficos de CPU/Memory
3. **Comparação Automática**: Compara cada scan com baseline e detecta desvios
4. **Término Automático**: Para automaticamente ao fim da duração configurada
5. **Relatório Final**: Gerado e exibido automaticamente ao término

### Relatório Final (ViewStressReport)

Exibido automaticamente ao término do teste:

- **Badge PASS/FAIL**: Verde (PASS) se <10% de HPAs com problemas críticos
- **Barra de Saúde**: Visualização percentual de HPAs saudáveis
- **Resumo Executivo**:
  - Duração total, número de scans, HPAs monitorados
  - Contagem de problemas (Critical/Warning/Info)
- **Métricas de Pico**:
  - **CPU Máximo**: valor, HPA, horário (GMT-3)
  - **Memory Máximo**: valor, HPA, horário
  - **Evolução de Réplicas**: `PRE → PEAK → POST` com % de aumento
    - Exemplo: `100 réplicas → 150 réplicas → 120 réplicas (+50, +50%)`
  - Taxa de Erro Máxima (se aplicável)
  - Latência P95 Máxima (se aplicável)
- **Problemas Detectados**:
  - Top 5 Critical Issues
  - Top 5 Warnings
  - Para cada: tipo, HPA afetado, descrição
- **Recomendações**:
  - Priorizadas (URGENTE/ALTO/MÉDIO/BAIXO)
  - Categorizadas (Scaling/Resources/Config/Code/Infra)
  - Ação sugerida + rationale

### Fluxo do Teste

```
1. Setup → Escolha modo "Stress Test"
2. Baseline Capture → Coleta 30min de histórico
3. Teste Inicia → Aplique carga externa
4. Monitoramento → Dashboard em tempo real
5. Término → Automático (duração) ou manual (Q)
6. Relatório → Exibido automaticamente com resultado PASS/FAIL
```

### Controles Especiais

- **P**: Pausar/Retomar scan durante teste
- **Shift+R**: Reiniciar teste (limpa dados, recaptura baseline, mantém na view)
- **E**: Exportar relatório em Markdown (TODO)
- **Shift+E**: Exportar relatório em PDF (TODO)

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

### Fase 1: Fundação ✅ (Concluída)
- [x] Setup projeto
- [x] Modelos de dados (HPASnapshot, TimeSeriesData, StressTestMetrics)
- [x] Armazenamento em memória com estatísticas
- [x] Detector de anomalias (10 tipos: Fase 1 + Fase 2)
- [x] Testes unitários abrangentes
- [x] Documentação completa

### Fase 2: Integração ✅ (Concluída)
- [x] Integração cliente K8s
- [x] Integração cliente Prometheus com port-forward automático
- [x] Coletor unificado (K8s + Prometheus + Analyzer)
- [x] Loop de monitoramento com canais
- [x] Sistema de configuração YAML
- [x] Persistência SQLite (auto-save/load/cleanup)

### Fase 3: Interface do Usuário ✅ (Concluída)
- [x] TUI completa com 7 views (Bubble Tea + Lipgloss)
- [x] Dashboard multi-cluster
- [x] View de alertas com filtragem
- [x] View de histórico com gráficos temporais (GMT-3)
- [x] **Modo Stress Test** com baseline e relatório automático
- [x] Scroll inteligente em menus grandes
- [x] Restart automático de testes

### Fase 4: Recursos Avançados 🔄 (Em Progresso)
- [x] Motor de correlação de alertas
- [x] Detecção avançada de anomalias
- [ ] Gestão de silêncios via TUI (Alertmanager)
- [ ] Descoberta automática (clusters, Prometheus, Alertmanager)
- [x] Persistência SQLite com retenção de 24h
- [ ] Exportação de relatórios (Markdown/PDF)

### Fase 5: Pronto para Produção 🔄 (Próxima)
- [ ] Arquivo de serviço systemd
- [ ] Imagem Docker oficial
- [ ] Notificações via webhook (Slack, Discord, Teams)
- [ ] Otimização de performance
- [ ] Testes de integração end-to-end
- [ ] Pipeline de CI/CD

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

**Status**: 🟢 Em desenvolvimento ativo - Fase 3 concluída (TUI + Stress Test)
