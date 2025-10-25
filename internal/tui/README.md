# HPA Watchdog - Terminal UI (TUI)

Interface de terminal rica e interativa para monitoramento de HPAs construída com [Bubble Tea](https://github.com/charmbracelet/bubbletea) e [Lipgloss](https://github.com/charmbracelet/lipgloss).

## 🎨 Features

- ✅ **4 Views Interativas**: Dashboard, Alertas, Clusters, Detalhes
- ✅ **Navegação por Teclado**: Tab, ↑↓, jk, Enter, filtros numéricos
- ✅ **Filtros Dinâmicos**: Por severidade (Critical, Warning, Info) e cluster
- ✅ **Atualização em Tempo Real**: Recebe dados via canais (Bubble Tea pattern)
- ✅ **Responsive**: Adapta ao tamanho do terminal
- ✅ **Colorido e Semântico**: Badges, métricas, tabelas com cores
- ✅ **Informativo**: Ações sugeridas, métricas detalhadas, histórico

## 📊 Views

### 1. Dashboard 📊
Vista geral do sistema com:
- Métricas gerais (clusters, HPAs, anomalias)
- Top 5 tipos de anomalias mais frequentes
- Top 5 clusters com mais anomalias
- 5 anomalias mais recentes

### 2. Alertas 🔔
Lista completa de anomalias com:
- Filtros interativos (1-4 para severidades)
- Tabela scrollable (↑↓)
- Colunas: Hora, Severidade, Tipo, HPA, Mensagem
- Seleção com Enter para ver detalhes

### 3. Clusters 🏢
Lista de todos os clusters:
- Status (Online/Offline/Error)
- Contadores de HPAs e anomalias
- Timestamp do último scan

### 4. Detalhes 🔍
Informações completas da anomalia selecionada:
- Localização (Cluster, Namespace, HPA)
- Métricas do HPA (Réplicas, CPU, Memory, Error Rate, Latency)
- Ações sugeridas (lista numerada)
- Mensagem completa com text wrapping

## ⌨️ Controles

| Tecla | Ação |
|-------|------|
| `Tab` | Próxima view (Dashboard → Alerts → Clusters → Details) |
| `Shift+Tab` | View anterior |
| `↑` ou `k` | Navegar para cima |
| `↓` ou `j` | Navegar para baixo |
| `Enter` | Selecionar item |
| `1` | Filtro: All |
| `2` | Filtro: Critical |
| `3` | Filtro: Warning |
| `4` | Filtro: Info |
| `R` ou `F5` | Force refresh |
| `Q` ou `Ctrl+C` | Sair |

## 🎨 Paleta de Cores

```go
ColorPrimary   = "#00D9FF" // Cyan brilhante
ColorSecondary = "#7C3AED" // Purple
ColorSuccess   = "#10B981" // Green
ColorWarning   = "#F59E0B" // Orange
ColorDanger    = "#EF4444" // Red
ColorInfo      = "#3B82F6" // Blue
ColorMuted     = "#6B7280" // Gray
```

## 🏗️ Arquitetura

### Model (Bubble Tea)
```go
type Model struct {
    currentView ViewType
    snapshots   map[string]*models.TimeSeriesData
    anomalies   []analyzer.Anomaly
    clusters    map[string]*ClusterInfo

    // Canais para receber dados
    snapshotChan chan *models.HPASnapshot
    anomalyChan  chan analyzer.Anomaly
}
```

### Event Loop
1. **Init()**: Configura canais e ticker
2. **Update(msg)**: Processa eventos (teclas, dados, timer)
3. **View()**: Renderiza interface baseada na view atual

### Data Flow
```
Collector → snapshotChan → Model.handleSnapshot() → Update snapshots/clusters
Detector  → anomalyChan  → Model.handleAnomaly()  → Update anomalies
Ticker    → tickMsg      → Model.Update()         → Refresh timestamp
```

## 🧪 Testes

9 testes unitários cobrindo:
- ✅ Inicialização do model
- ✅ Manipulação de snapshots
- ✅ Manipulação de anomalias
- ✅ Filtros
- ✅ Renderização de views
- ✅ Helpers (truncate, wrapText, makeKey)

```bash
go test ./internal/tui/... -v
# PASS: 9/9 tests
```

## 🚀 Como Usar

### Integração com Collector

```go
import (
    "github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/tui"
    tea "github.com/charmbracelet/bubbletea"
)

// Cria TUI
model := tui.New()

// Inicia Bubble Tea
p := tea.NewProgram(model, tea.WithAltScreen())

// Goroutine do collector envia dados
go func() {
    for snapshot := range collectorChan {
        model.GetSnapshotChan() <- snapshot
    }
}()

// Goroutine do detector envia anomalias
go func() {
    for anomaly := range detectorChan {
        model.GetAnomalyChan() <- anomaly
    }
}()

// Executa TUI
if _, err := p.Run(); err != nil {
    log.Fatal(err)
}
```

### Teste Standalone

```bash
# Build
go build -o build/test-tui ./cmd/test-tui/main.go

# Run (requer terminal interativo)
./build/test-tui

# Ou use o script
./test-tui.sh
```

## 📁 Estrutura de Arquivos

```
internal/tui/
├── README.md              # Esta documentação
├── styles.go              # Paleta de cores e estilos Lipgloss
├── model.go               # Model Bubble Tea principal
├── view_dashboard.go      # View Dashboard
├── view_alerts.go         # View Alertas
├── view_clusters.go       # View Clusters
├── view_details.go        # View Detalhes
└── tui_test.go            # Testes unitários (9 tests)
```

## 🔮 Próximos Passos

- [ ] Gráficos ASCII de métricas (usando asciigraph)
- [ ] Exportar anomalias para arquivo
- [ ] Silence management (integração com Alertmanager)
- [ ] Navegação drill-down (Cluster → Namespace → HPA)
- [ ] Search/filter por nome de HPA
- [ ] Themes (dark, light, monokai)

## 📝 Notas

- **TTY Required**: A TUI precisa de um terminal interativo (não funciona em ambientes sem TTY como CI/CD)
- **Performance**: Otimizada para até 100 clusters e 1000 anomalias simultâneas
- **Thread-Safe**: Todos os acessos a dados compartilhados usam canais (Bubble Tea pattern)
- **Responsive**: Usa `WindowSizeMsg` para adaptar ao resize do terminal

## 🎯 Exemplos de Uso

### Monitoramento Multi-Cluster
```bash
# Terminal 1: Collector rodando
./hpa-watchdog

# Terminal mostra:
# - 24 clusters online
# - 2.400 HPAs monitorados
# - 15 anomalias ativas
# - Top clusters: cluster-prd-1 (8 anomalias), cluster-prd-2 (5 anomalias)
```

### Troubleshooting de Anomalia
```bash
# 1. Dashboard mostra "CPU_SPIKE" no cluster-api-prd
# 2. Tab → Alerts view
# 3. ↓ para selecionar anomalia
# 4. Enter para ver detalhes
# 5. View Details mostra:
#    - CPU: 45% → 95% (+111%)
#    - Ações sugeridas:
#      1. Verificar aumento de tráfego
#      2. Verificar logs para slow queries
#      3. Monitorar scaling do HPA
```

## 🐛 Troubleshooting

### "could not open a new TTY"
- **Causa**: Ambiente sem TTY (WSL, SSH sem -t, CI/CD)
- **Solução**: Execute em terminal interativo local ou use `ssh -t`

### Views não atualizam
- **Causa**: Canais bloqueados ou ticker parado
- **Solução**: Verifique que goroutines do collector/detector estão rodando

### Cores não aparecem
- **Causa**: Terminal não suporta cores ou TERM não configurado
- **Solução**: Export `TERM=xterm-256color`

## 📚 Referências

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Framework TUI
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling para terminal
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
