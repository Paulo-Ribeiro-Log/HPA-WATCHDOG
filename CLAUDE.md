# CLAUDE.md

Este arquivo fornece orientações ao Claude Code (claude.ai/code) ao trabalhar com o código neste repositório.

## Visão Geral do Projeto

**HPA Watchdog** é um sistema autônomo de monitoramento de Horizontal Pod Autoscalers (HPAs) do Kubernetes em múltiplos clusters. Ele oferece uma TUI (Terminal UI) rica construída com Bubble Tea e Lipgloss, fornecendo monitoramento em tempo real, detecção de anomalias e gerenciamento centralizado de alertas.

**Status**: 🟢 Fase de desenvolvimento - Componentes principais implementados
**Meta**: Monitoramento de HPAs multi-cluster com integração Prometheus + Alertmanager

### Status da Implementação
- ✅ **Camada de Armazenamento**: Cache de séries temporais em memória + persistência em SQLite (retenção de 24h)
- ✅ **Camada de Análise**: Fase 1 (estado persistente) + Fase 2 (mudanças súbitas) - 10 tipos de anomalia
- ✅ **Camada de Cliente K8s**: Coleta de HPA e criação de snapshots
- ✅ **Camada de Cliente Prometheus**: Enriquecimento de métricas com consultas PromQL
- ✅ **Camada de Coleta**: Orquestração unificada de K8s + Prometheus + Analyzer
- ✅ **Camada de Configuração**: Sistema de configuração baseado em YAML
- ✅ **Camada de Persistência**: SQLite com auto-save/load e limpeza
- ✅ **Camada TUI**: 7 views implementadas (Dashboard, Alertas, Clusters, Histórico, Stress Test, Relatório Final, Detalhes)
- ✅ **Stress Test Mode**: Baseline capture, comparação em tempo real, relatório final automatizado
- ⚠️ **Camada Alertmanager**: Opcional (não crítica para o MVP)

## Filosofia Central: KISS (Keep It Simple, Stupid)

**IMPORTANTE**: Este projeto segue o princípio KISS à risca. Ao desenvolver:

- **Prefira simplicidade a esperteza** - Código direto supera soluções "inteligentes"
- **Evite superengenharia** - Construa o que é necessário agora, não o que pode ser necessário depois
- **Evite otimização prematura** - Faça funcionar primeiro, otimize apenas se for realmente necessário
- **Use tecnologia entediante** - Bibliotecas comprovadas em vez de novidades/ tendências
- **Claro em vez de conciso** - Código legível vence código curto
- **Uma responsabilidade por componente** - Cada módulo deve fazer uma coisa bem
- **Falhe rápido e de forma evidente** - Melhor travar com um erro claro do que falhar silenciosamente
- **Configuração antes de código** - Prefira tornar o comportamento configurável a codificar lógica complexa fixa

### KISS na Prática

- **Loop de monitoramento**: Goroutine simples por cluster, sem agendamento complexo
- **Armazenamento de dados**: Abordagem híbrida - RAM (acesso rápido de 5min) + SQLite (persistência de 24h)
- **Correlação de alertas**: Agrupamento básico por cluster/namespace/HPA - sem complexidade de ML/IA
- **TUI**: Padrões padrão do Bubble Tea - nada de frameworks personalizados
- **Tratamento de erros**: Mensagens claras, degradação graciosa - sem falhas silenciosas
- **Persistência**: Auto-save para SQLite (assíncrono), auto-load no startup, auto-cleanup de dados antigos

Se uma solução parecer complexa, provavelmente é. Dê um passo atrás e encontre a abordagem mais simples.

## Arquitetura

### Coleta de Dados em Três Camadas

1. **Kubernetes API** (client-go): Configuração do HPA, contagem de réplicas, informações de deployment, eventos
2. **Prometheus API**: Métricas (CPU/Memória/Rede) e análise temporal com PromQL
3. **Alertmanager API**: Agregação de alertas existentes e gerenciamento de silêncios

### Abordagem Híbrida

- **API do K8s**: Dados de configuração e estado (réplicas mín./máx., réplicas atuais/desejadas)
- **Prometheus**: Fonte principal de métricas com histórico nativo e métricas ricas (CPU, Memória, taxa de requisições, taxa de erros, latência P95)
- **Alertmanager**: Fonte principal de alertas vindos de regras Prometheus existentes (70% dos alertas)
- **Watchdog Analyzer**: Detecção de anomalias complementar para padrões não cobertos pelas consultas PromQL simples (30% dos alertas)

## Modelo de Dados Central

### HPASnapshot
Snapshot estendido que captura tanto o estado do K8s quanto métricas do Prometheus:
- Dados do K8s: Configuração do HPA, réplicas, requests/limits de recursos, status
- Dados do Prometheus: Métricas atuais, histórico de 5 minutos, métricas estendidas (taxa de requisições, taxa de erros, latência)
- Indicador de fonte de dados: Prometheus (preferencial), Metrics-Server (fallback) ou híbrido

### UnifiedAlert
Combina alertas do Alertmanager e da detecção própria do Watchdog:
- Rastreamento da origem (Alertmanager vs Watchdog)
- Enriquecimento com HPASnapshot e AlertContext
- Correlação com alertas relacionados
- Suporte a silêncio e confirmação

## Estrutura do Projeto

```
hpa-watchdog/
├── cmd/
│   └── main.go                    # Ponto de entrada
├── internal/
│   ├── analyzer/                  # ✅ IMPLEMENTADO
│   │   ├── detector.go            # Detector de anomalias com 10 tipos (Fase 1 + Fase 2)
│   │   ├── detector_test.go       # 12 testes unitários (Fase 1)
│   │   ├── sudden_changes_test.go # 8 testes unitários (Fase 2)
│   │   └── README.md              # Documentação
│   ├── storage/                   # ✅ IMPLEMENTADO
│   │   ├── cache.go               # Cache de séries temporais com integração de persistência
│   │   ├── cache_test.go          # 12 testes do cache
│   │   ├── persistence.go         # Camada de persistência com SQLite
│   │   ├── persistence_test.go    # 8 testes de persistência
│   │   └── README.md              # Documentação
│   ├── models/                    # ✅ IMPLEMENTADO
│   │   └── types.go               # HPASnapshot, TimeSeriesData, HPAStats, GetPrevious()
│   ├── monitor/                   # 🔄 TODO
│   │   ├── collector.go           # Coletor unificado (K8s + Prometheus + Alertmanager)
│   │   ├── analyzer.go            # Detecção de anomalias
│   │   └── alerter.go             # Sistema de alertas
│   ├── prometheus/                # 🔄 TODO
│   │   ├── client.go              # Wrapper da API do Prometheus
│   │   ├── queries.go             # Consultas PromQL predefinidas
│   │   └── discovery.go           # Descoberta automática de endpoints
│   ├── alertmanager/              # 🔄 TODO
│   │   └── client.go              # Wrapper da API do Alertmanager
│   ├── config/                    # 🔄 TODO
│   │   ├── loader.go              # Carregamento de configuração
│   │   ├── thresholds.go          # Gerenciamento de thresholds
│   │   └── clusters.go            # Descoberta de clusters
│   └── tui/                       # 🔄 TODO
│       ├── app.go                 # Aplicativo principal Bubble Tea
│       ├── views.go               # Renderização das views
│       ├── handlers.go            # Manipuladores de eventos
│       ├── components/            # Componentes de UI (dashboard, alertas, gráficos, config)
│       └── styles.go              # Estilos do Lipgloss
├── configs/
│   └── watchdog.yaml              # Configuração padrão
└── HPA_WATCHDOG_*.md              # Documentos de especificação
```

## Comandos de Desenvolvimento

### Build
```bash
# Compilar o binário
go build -o build/hpa-watchdog ./cmd/main.go

# Compilar com informação de versão
go build -ldflags "-X main.Version=v1.0.0" -o build/hpa-watchdog ./cmd/main.go
```

### Execução
```bash
# Executar com a configuração padrão
./build/hpa-watchdog

# Executar com uma configuração personalizada
./build/hpa-watchdog --config /caminho/para/watchdog.yaml

# Modo debug (logs verbosos)
./build/hpa-watchdog --debug
```

### Testes
```bash
# Executar todos os testes
go test ./...

# Testar um pacote específico
go test ./internal/monitor/...

# Testar com cobertura
go test -cover ./...

# Testes de integração (exige acesso a um cluster K8s)
go test ./tests/integration/...
```

### Validação da Configuração
```bash
# Validar arquivo de configuração
./build/hpa-watchdog validate --config configs/watchdog.yaml
```

## Dependências Principais

- **k8s.io/client-go@v0.31.4**: Cliente da API do Kubernetes
- **github.com/charmbracelet/bubbletea@v0.24.2**: Framework de TUI
- **github.com/charmbracelet/lipgloss@v1.1.0**: Estilização de terminal
- **github.com/prometheus/client_golang**: Cliente da API do Prometheus
- **github.com/spf13/viper**: Gerenciamento de configuração
- **github.com/guptarohit/asciigraph**: Gráficos ASCII para métricas
- **github.com/rs/zerolog**: Logging estruturado
- **github.com/mattn/go-sqlite3**: Persistência em SQLite (necessária em produção)

## Consultas Importantes de Prometheus

### Uso de CPU (alvo do HPA)
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}",pod=~"{pod_selector}"}[1m])) /
sum(kube_pod_container_resource_requests{namespace="{namespace}",pod=~"{pod_selector}",resource="cpu"}) * 100
```

### Histórico de Réplicas
```promql
kube_horizontalpodautoscaler_status_current_replicas{namespace="{namespace}",horizontalpodautoscaler="{name}"}[5m]
```

### Taxa de Requisições
```promql
sum(rate(http_requests_total{namespace="{namespace}",service="{service}"}[1m]))
```

### Taxa de Erros
```promql
sum(rate(http_requests_total{namespace="{namespace}",service="{service}",status=~"5.."}[1m])) /
sum(rate(http_requests_total{namespace="{namespace}",service="{service}"}[1m])) * 100
```

## Sistema de Configuração

### Arquivo de Configuração: `configs/watchdog.yaml`

Seções principais:
- **monitoring**: Intervalos de varredura, definições de Prometheus/Alertmanager, descoberta automática
- **clusters**: Descoberta e filtragem de clusters
- **storage**: Persistência opcional com SQLite
- **alerts**: Prioridade da fonte, deduplicação, correlação
- **thresholds**: Limites de CPU/Memória, deltas de réplicas, métricas estendidas
- **ui**: Taxa de atualização, tema, sons

### Descoberta Automática

- **Clusters**: Descobre a partir do kubeconfig ou `clusters-config.json`
- **Prometheus**: Testa padrões comuns de serviço no namespace de monitoramento
- **Alertmanager**: Testa padrões comuns de serviço no namespace de monitoramento
- **Fallback**: Usa o Metrics-Server do Kubernetes se o Prometheus não estiver disponível

## Estratégia de Persistência de Dados

### Armazenamento Híbrido: RAM + SQLite ✅

**Por que híbrido?**
- **RAM (5min)**: Acesso ultrarrápido para comparações e detecção de anomalias
- **SQLite (24h)**: Persistência que sobrevive a reinicializações e permite análise histórica

### Implementação (`internal/storage/`)

#### Cache em Memória (TimeSeriesCache)
```go
cache := storage.NewTimeSeriesCache(&CacheConfig{
    MaxDuration:  5 * time.Minute,  // Janela deslizante
    ScanInterval: 30 * time.Second, // ~10 snapshots por HPA
})
```

- **Acesso rápido**: Busca O(1) por cluster/namespace/nome
- **Limpeza automática**: Remove snapshots com mais de 5 minutos
- **Estatísticas**: Tendências pré-calculadas de CPU/Memória, variações de réplicas
- **Thread-safe**: sync.RWMutex para acesso concorrente

#### Persistência em SQLite
```go
persist, _ := storage.NewPersistence(&PersistenceConfig{
    Enabled:     true,
    DBPath:      "~/.hpa-watchdog/snapshots.db",
    MaxAge:      24 * time.Hour,
    AutoCleanup: true,
})

cache.SetPersistence(persist)  // Auto-save habilitado!
```

**Recursos**:
- **Auto-save**: Cada snapshot adicionado ao cache é salvo em SQLite (assíncrono)
- **Auto-load**: No startup, carrega os últimos 5 minutos do SQLite para a RAM
- **Auto-cleanup**: Remove snapshots com mais de 24h
- **Operações em lote**: Inserts/consultas em massa eficientes
- **Schema**: Tabela simples com serialização JSON dos snapshots

**Schema do Banco**:
```sql
CREATE TABLE snapshots (
    cluster TEXT,
    namespace TEXT,
    hpa_name TEXT,
    timestamp DATETIME,
    data TEXT  -- HPASnapshot completo como JSON
)
```

**Estimativas de armazenamento** (24 clusters, 2400 HPAs):
- Memória: ~12 MB (janela de 5min)
- SQLite: ~3,3 GB (retenção de 24h, auto-cleanup)
- Tempo de varredura: <5s por cluster (2880 varreduras/dia)

### Benefícios da Persistência para Multi-Cluster

1. **Sobrevive a reinicializações**: Sem perda de dados quando o HPA Watchdog reinicia
2. **Detecção imediata**: Detecta mudanças súbitas desde o primeiro scan (carrega estado anterior)
3. **Análise histórica**: 24h de dados para análise de tendências e depuração
4. **Baixo uso de memória**: Apenas 5min em RAM, restante no SQLite
5. **Performance**: Saves assíncronos não bloqueiam o loop de monitoramento

## Loop de Monitoramento

Cada cluster executa uma goroutine independente:
1. Lista namespaces (pula namespaces de sistema)
2. Para cada namespace, lista HPAs
3. Para cada HPA:
   - Obtém a configuração via API do K8s
   - Consulta métricas no Prometheus (atual + histórico de 5min)
   - Cria o HPASnapshot
   - Armazena no cache de séries temporais → **Auto-salvo no SQLite**
4. Sincroniza alertas do Alertmanager
5. Analisa snapshots em busca de anomalias (persistentes e súbitas)
6. Envia alertas unificados para a TUI via canais
7. Dorme até o próximo intervalo de varredura

**Na inicialização**: Carrega os últimos 5 minutos do SQLite → Pronto para detectar mudanças imediatamente!

## Modo Stress Test

O HPA Watchdog possui um modo especializado para testes de carga e validação de configurações de HPA:

### Funcionalidades
1. **Baseline Capture**: Captura estado PRE (réplicas, CPU, memory) antes do teste iniciar
2. **Monitoramento em Tempo Real**: Dashboard interativo com gráficos de CPU/Memory (timezone GMT-3)
3. **Comparação Automática**: Compara cada scan com baseline e detecta desvios
4. **Término Automático**: Para automaticamente ao fim da duração configurada
5. **Relatório Final Automático**: Gera e exibe relatório completo ao término

### Fluxo do Stress Test
```
Setup → Baseline Capture (30min histórico) → Teste Inicia → Scans Periódicos
→ Comparação com Baseline → Término (automático ou manual) → Relatório Final
```

### Relatório Final
Gerado automaticamente ao término e exibido na **ViewStressReport**:
- **Badge PASS/FAIL**: Baseado em % de HPAs com problemas críticos (<10% = PASS)
- **Barra de Saúde**: Visualização percentual de HPAs saudáveis
- **Resumo Executivo**: Duração, scans, HPAs monitorados, problemas detectados
- **Métricas de Pico**:
  - CPU Máximo (valor, HPA, horário)
  - Memory Máximo (valor, HPA, horário)
  - **Evolução de Réplicas**: PRE → PEAK → POST com % de aumento
  - Taxa de Erro Máxima (se aplicável)
  - Latência P95 Máxima (se aplicável)
- **Problemas Detectados**: Lista de Critical Issues e Warnings (top 5 cada)
- **Recomendações**: Ações priorizadas por categoria (Scaling/Resources/Config/Code/Infra)

### Controles do Stress Test
- **P**: Pausar/Retomar scan
- **Shift+R**: Reiniciar teste (mantém na view, limpa dados, recaptura baseline)
- **E**: Exportar relatório em Markdown (TODO)
- **Shift+E**: Exportar relatório em PDF (TODO)
- **Scroll**: Menu de seleção de HPAs com viewport para listas grandes

### StressTestMetrics
Estrutura completa (`internal/models/stresstest.go`) que captura:
- Metadados do teste (nome, duração, status, scans)
- Métricas gerais (clusters, HPAs, problemas)
- Métricas de pico (PeakMetrics struct)
- Problemas por severidade (CriticalIssues, WarningIssues, InfoIssues)
- HPAMetrics por HPA individual
- Timeline de eventos
- Recomendações geradas

**Persistência**: Baseline e resultados são salvos no SQLite para análise posterior.

## Detecção de Anomalias

### Integração com Alertmanager (Primária)
- Sincroniza alertas existentes via API do Alertmanager
- Filtra alertas relacionados a HPA
- Enriquece com contexto (métricas, histórico, correlação)
- Fornece visão centralizada multi-cluster
- Permite gerenciar silêncios diretamente pela TUI

### Watchdog Analyzer - Fase 1: Anomalias de Estado Persistente ✅
O pacote analyzer (`internal/analyzer/`) implementa 5 detectores para estados problemáticos persistentes:

| # | Anomalia | Condição | Duração | Status |
|---|----------|----------|---------|--------|
| 1 | **Oscilação** | >5 alterações de réplica | 5min | ✅ Implementado |
| 2 | **No Limite** | réplicas = máx + CPU > alvo +20% | 2min | ✅ Implementado |
| 3 | **OOMKilled** | Pod finalizado por OOM | - | 🔴 Placeholder |
| 4 | **Pods Não Prontos** | Pods não prontos | 3min | ✅ Implementado |
| 5 | **Alta Taxa de Erros** | >5% de erros 5xx (Prometheus) | 2min | ✅ Implementado |

**Testes**: 12/12 testes unitários aprovados (veja `internal/analyzer/detector_test.go`)

### Watchdog Analyzer - Fase 2: Mudanças Súbitas ✅
Detecta variações bruscas entre scans consecutivos (comparação scan a scan):

| # | Anomalia | Condição | Limite | Status |
|---|----------|----------|--------|--------|
| 6 | **Pico de CPU** | CPU aumentou >50% em 1 scan | +50% | ✅ Implementado |
| 7 | **Pico de Réplicas** | Réplicas aumentaram em 1 scan | +3 | ✅ Implementado |
| 8 | **Pico de Erros** | Taxa de erros aumentou em 1 scan | +5% | ✅ Implementado |
| 9 | **Pico de Latência** | Latência aumentou >100% em 1 scan | +100% | ✅ Implementado |
| 10 | **Queda de CPU** | CPU caiu >50% em 1 scan | -50% | ✅ Implementado |

**Principais características**:
- **Comparação scan a scan**: Compara o snapshot mais recente com o anterior (sem novas consultas ao Prometheus)
- **Detecção rápida**: Identifica mudanças súbitas imediatamente (dentro de um intervalo de varredura)
- **Cache local**: Usa `GetPrevious()` de TimeSeriesData para comparação instantânea
- **Thresholds configuráveis**: Todos os limites de picos são customizáveis
- **Sugestões de ação**: Cada anomalia inclui ações de remediação

**Testes**: 8/8 testes unitários aprovados (veja `internal/analyzer/sudden_changes_test.go`)

### Estratégia de Detecção Combinada
O analyzer executa as duas fases em cada varredura:
1. **Fase 1** detecta estados problemáticos persistentes (requer duração)
2. **Fase 2** detecta variações súbitas (requer 2 snapshots)

Total: **10 tipos de anomalia** cobrindo tanto tendências graduais quanto mudanças abruptas.

## Navegação da TUI

### Controles de Teclado
#### Gerais
- `Tab`: Troca de views (Dashboard, Alertas, Clusters, Histórico, Stress Test, Relatório)
- `↑↓` ou `j k`: Navega em listas (com scroll automático em menus grandes)
- `Enter`: Ver detalhes / Selecionar
- `H` ou `Home`: Volta para Dashboard
- `F5` ou `R`: Forçar refresh
- `Ctrl+C` ou `Q`: Sair
- `?`: Ajuda

#### Alertas
- `A`: Reconhecer alerta
- `Shift+A`: Reconhecer todos os alertas
- `S`: Silenciar alerta (cria silêncio no Alertmanager)
- `C`: Limpar alertas reconhecidos
- `E`: Enriquecer alerta com contexto de métricas
- `D`: Ver detalhes do alerta

#### Stress Test
- `P`: Pausar/Retomar scan
- `Shift+R`: Reiniciar teste automaticamente (mantém na view de stress test)
- `E`: Exportar relatório em Markdown
- `Shift+E`: Exportar relatório em PDF

### Visões (7 views implementadas)
1. **Setup**: Configuração inicial interativa (clusters, modo, duração, intervalo)
2. **Dashboard**: Visão geral multi-cluster, resumo de alertas, top clusters, anomalias recentes
3. **Alertas**: Lista detalhada de alertas com filtragem por severidade/cluster e correlação
4. **Clusters**: Detalhamento por cluster e namespace com métricas agregadas
5. **Histórico**: Análise temporal com gráficos de CPU/Memory/Réplicas (timezone GMT-3)
6. **Stress Test**: Dashboard em tempo real com baseline, gráficos de CPU/Memory, seleção de HPAs com scroll
7. **Relatório Final**: Resumo executivo do stress test (PASS/FAIL, métricas de pico PRE→PEAK→POST, recomendações)

## Correlação de Alertas

O Watchdog correlaciona automaticamente alertas relacionados:
- Agrupa alertas por cluster/namespace/HPA
- Identifica causa raiz vs sintomas
- Fornece análise combinada envolvendo múltiplos tipos de alerta
- Sugere ações de remediação

Exemplo: Pico de CPU → réplicas no limite → alta taxa de erros → alta latência correlacionados como um único incidente.

## Princípios de Design

1. **Segurança com runes**: Use sempre `[]rune` para lidar com texto Unicode na TUI
2. **Operações assíncronas**: Use comandos Bubble Tea para tarefas assíncronas (consultas K8s/Prometheus)
3. **Canais para atualizações**: Goroutines de monitoramento enviam updates para a TUI via canais
4. **Estratégia de fallback**: Prometheus → Metrics-Server, com degradação graciosa
5. **Armazenamento mínimo**: Aproveite o TSDB do Prometheus em vez de caches locais pesados
6. **Somente leitura**: Sem modificações no cluster, operações de monitoramento seguras

## Segurança e Permissões

### RBAC necessário no K8s
```yaml
- apiGroups: [""]
  resources: ["namespaces", "pods"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
  verbs: ["get", "list"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
```

**Observação**: Todas as operações são somente leitura. Nenhuma permissão de escrita/modificação é necessária.

## Metas de Desempenho

- **Tempo de varredura**: <5s por cluster (50 HPAs, 10 namespaces)
- **Uso de memória**: <100 MB (5 clusters, 250 HPAs, histórico de 5min)
- **Uso de CPU**: <5% em idle
- **Sincronização com Alertmanager**: Intervalo de 30s
- **Atualização da TUI**: 500ms

## Status do Roadmap

### Fase 1: Fundação ✅ (Concluída)
- ✅ Setup e estrutura do projeto
- ✅ Modelos de dados (HPASnapshot, TimeSeriesData, HPAStats)
- ✅ Armazenamento em memória com estatísticas
- ✅ Detector de anomalias (5 anomalias críticas)
- ✅ Testes unitários abrangentes (storage + analyzer)
- ✅ Documentação (README para cada pacote)

### Fase 2: Integração ✅ (Concluída)
- ✅ Integração com cliente K8s (`monitor/k8s_client.go`)
- ✅ Integração com cliente Prometheus (`prometheus/client.go`)
- ⚠️ Integração com Alertmanager (TODO - não crítico para o MVP)
- ✅ Coletor unificado (`monitor/collector.go`)
- ✅ Implementação do loop de monitoramento com canais
- ✅ Sistema de configuração com suporte YAML (`config/loader.go`)
- ✅ Todos os testes aprovados (analyzer, storage, monitor, prometheus)

### Fase 3: Interface do Usuário (Atual)
- 🔄 TUI básica (Bubble Tea)
- 🔄 Visão de dashboard (overview multi-cluster)
- 🔄 Visão de alertas (com filtragem)
- 🔄 Visão detalhada de cluster
- 🔄 Gráficos ASCII para métricas
- 🔄 Modal de configuração
- 🔄 Integração com canais do coletor

### Fase 4: Recursos Avançados
- 🔄 Motor de correlação de alertas
- 🔄 Gestão de silêncios via TUI
- 🔄 Detecção de anomalias aprimorada (anomalias da Fase 2)
- 🔄 Persistência SQLite (opcional)
- 🔄 Descoberta automática (clusters, Prometheus, Alertmanager)

### Fase 5: Pronto para Produção
- 🔄 Arquivo de serviço systemd
- 🔄 Imagem Docker
- 🔄 Notificações via webhook (Slack, Discord, Teams)
- 🔄 Otimização de performance
- 🔄 Testes de integração
- 🔄 Pipeline de CI/CD

## Padrões Comuns

### Adicionando um Novo Tipo de Anomalia
1. Adicione a constante de tipo de anomalia em `internal/analyzer/detector.go` (`AnomalyType`)
2. Adicione a configuração de threshold na struct `DetectorConfig`
3. Implemente o método de detecção (ex.: `detectNewAnomaly()`)
4. Chame o método de detecção no loop `Detect()`
5. Adicione testes unitários em `internal/analyzer/detector_test.go`
6. Atualize o README com os detalhes da nova anomalia

### Adicionando uma Nova Consulta Prometheus
1. Defina o template da consulta em `internal/prometheus/queries.go`
2. Adicione a lógica de parsing para o formato do resultado
3. Integre ao coletor em `internal/monitor/collector.go`
4. Atualize o modelo `HPASnapshot` se precisar de um novo campo

### Expandindo Views da TUI
1. Crie o componente em `internal/tui/components/`
2. Implemente os métodos `Model`, `Update` e `View` do Bubble Tea
3. Integre no app principal em `internal/tui/app.go`
4. Adicione handlers de teclado em `internal/tui/handlers.go`
5. Defina estilos em `internal/tui/styles.go`

## Integração com k8s-hpa-manager

Embora o HPA Watchdog possa compartilhar código utilitário com o projeto k8s-hpa-manager (descoberta de clusters, wrappers do cliente K8s), ele é **completamente autônomo**:
- Binário separado: `hpa-watchdog`
- Diretório de configuração separado: `~/.hpa-watchdog/`
- Operação independente (não exige que o k8s-hpa-manager esteja rodando)
- Pode rodar como daemon em background ou TUI interativa

## Solução de Problemas

### Problemas de Conexão com o Prometheus
- Verifique o endpoint: `kubectl port-forward -n monitoring svc/prometheus 9090:9090`
- Cheque os padrões de descoberta automática na configuração
- Habilite o fallback para metrics-server: `prometheus.fallback_to_metrics_server: true`

### Métricas Ausentes
- Garanta que o Prometheus está coletando kube-state-metrics
- Verifique se o metrics-server está instalado: `kubectl top pods`
- Confirme que as métricas alvo do HPA estão expostas

### Alto Uso de Memória
- Reduza `history_retention_minutes` (padrão: 5)
- Limite `max_active_alerts` (padrão: 100)
- Desabilite a persistência se não for necessária

### Problemas de Sincronização com o Alertmanager
- Verifique a acessibilidade do endpoint do Alertmanager
- Cheque os filtros de labels dos alertas: `filters.only_hpa_related: true`
- Aumente o intervalo de sync se houver rate-limiting
- "As mensagens de commit devem ser sempre em pt-br"
- "O claude.md deve ser sempre em pt-br"
