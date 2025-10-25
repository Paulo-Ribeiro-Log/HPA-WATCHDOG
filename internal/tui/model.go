package tui

import (
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewType define os tipos de views disponíveis
type ViewType int

const (
	ViewSetup ViewType = iota // Setup inicial (NOVA VIEW)
	ViewDashboard
	ViewAlerts
	ViewClusters
	ViewDetails
)

// Model é o model principal da aplicação Bubble Tea
type Model struct {
	// Estado da UI
	currentView ViewType
	width       int
	height      int
	ready       bool

	// Setup state (para configuração inicial)
	setupState *SetupState

	// Dados
	snapshots map[string]*models.TimeSeriesData // cluster/namespace/name -> TimeSeriesData
	anomalies []analyzer.Anomaly
	clusters  map[string]*ClusterInfo // cluster -> info

	// Navegação
	selectedCluster   string
	selectedNamespace string
	selectedHPA       string
	selectedAnomaly   int
	cursorPos         int

	// Filtros
	filterSeverity string // "All", "Critical", "Warning", "Info"
	filterCluster  string // "" = All

	// Estado
	lastUpdate    time.Time
	autoRefresh   bool
	refreshTicker *time.Ticker

	// Canais de dados (recebe atualizações do monitor)
	snapshotChan chan *models.HPASnapshot
	anomalyChan  chan analyzer.Anomaly
}

// ClusterInfo informações resumidas de um cluster
type ClusterInfo struct {
	Name           string
	TotalHPAs      int
	TotalAnomalies int
	Status         string // "Online", "Offline", "Error"
	LastScan       time.Time
}

// New cria nova instância do model
func New() Model {
	return Model{
		currentView:    ViewSetup, // Inicia com setup
		setupState:     NewSetupState(),
		snapshots:      make(map[string]*models.TimeSeriesData),
		anomalies:      []analyzer.Anomaly{},
		clusters:       make(map[string]*ClusterInfo),
		filterSeverity: "All",
		filterCluster:  "",
		autoRefresh:    true,
		snapshotChan:   make(chan *models.HPASnapshot, 100),
		anomalyChan:    make(chan analyzer.Anomaly, 100),
	}
}

// Init inicializa o model (Bubble Tea)
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForSnapshot(m.snapshotChan),
		waitForAnomaly(m.anomalyChan),
	)
}

// Update processa mensagens (Bubble Tea)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tickMsg:
		m.lastUpdate = time.Now()
		return m, tickCmd()

	case snapshotMsg:
		m.handleSnapshot(msg.snapshot)
		return m, waitForSnapshot(m.snapshotChan)

	case anomalyMsg:
		m.handleAnomaly(msg.anomaly)
		return m, waitForAnomaly(m.anomalyChan)
	}

	return m, nil
}

// View renderiza a UI (Bubble Tea)
func (m Model) View() string {
	if !m.ready {
		return "Inicializando HPA Watchdog..."
	}

	switch m.currentView {
	case ViewSetup:
		return m.renderSetup()
	case ViewDashboard:
		return m.renderDashboard()
	case ViewAlerts:
		return m.renderAlerts()
	case ViewClusters:
		return m.renderClusters()
	case ViewDetails:
		return m.renderDetails()
	default:
		return "View desconhecida"
	}
}

// handleKeyPress processa teclas pressionadas
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Setup tem controles próprios
	if m.currentView == ViewSetup {
		return m.handleSetupKeyPress(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		m.currentView = (m.currentView + 1) % 5 // 5 views agora (Setup incluído)
		m.cursorPos = 0
		return m, nil

	case "shift+tab":
		m.currentView = (m.currentView + 4) % 5 // -1 com wrap
		m.cursorPos = 0
		return m, nil

	case "up", "k":
		if m.cursorPos > 0 {
			m.cursorPos--
		}
		return m, nil

	case "down", "j":
		maxPos := m.getMaxCursorPos()
		if m.cursorPos < maxPos {
			m.cursorPos++
		}
		return m, nil

	case "enter":
		return m.handleSelect()

	case "r", "f5":
		// Force refresh
		return m, nil

	case "1", "2", "3", "4":
		// Filtro de severidade
		switch msg.String() {
		case "1":
			m.filterSeverity = "All"
		case "2":
			m.filterSeverity = "Critical"
		case "3":
			m.filterSeverity = "Warning"
		case "4":
			m.filterSeverity = "Info"
		}
		m.cursorPos = 0
		return m, nil
	}

	return m, nil
}

// handleSnapshot processa novo snapshot
func (m *Model) handleSnapshot(snapshot *models.HPASnapshot) {
	key := makeKey(snapshot.Cluster, snapshot.Namespace, snapshot.Name)

	// Cria ou atualiza TimeSeriesData
	ts, exists := m.snapshots[key]
	if !exists {
		ts = &models.TimeSeriesData{
			HPAKey:      key,
			Snapshots:   []models.HPASnapshot{},
			MaxDuration: 5 * time.Minute,
		}
		m.snapshots[key] = ts
	}

	ts.Add(*snapshot)

	// Atualiza info do cluster
	cluster, exists := m.clusters[snapshot.Cluster]
	if !exists {
		cluster = &ClusterInfo{
			Name:   snapshot.Cluster,
			Status: "Online",
		}
		m.clusters[snapshot.Cluster] = cluster
	}

	cluster.TotalHPAs++
	cluster.LastScan = time.Now()
}

// handleAnomaly processa nova anomalia
func (m *Model) handleAnomaly(anomaly analyzer.Anomaly) {
	// Adiciona ao início da lista (mais recente primeiro)
	m.anomalies = append([]analyzer.Anomaly{anomaly}, m.anomalies...)

	// Limita a 100 anomalias
	if len(m.anomalies) > 100 {
		m.anomalies = m.anomalies[:100]
	}

	// Atualiza contador do cluster
	if cluster, exists := m.clusters[anomaly.Cluster]; exists {
		cluster.TotalAnomalies++
	}
}

// handleSelect processa seleção de item
func (m Model) handleSelect() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewAlerts:
		if m.cursorPos < len(m.getFilteredAnomalies()) {
			m.selectedAnomaly = m.cursorPos
			m.currentView = ViewDetails
		}
	case ViewClusters:
		// Seleciona cluster
		clusters := m.getSortedClusters()
		if m.cursorPos < len(clusters) {
			m.selectedCluster = clusters[m.cursorPos].Name
			// Futuramente pode abrir view detalhada do cluster
		}
	}
	return m, nil
}

// getMaxCursorPos retorna posição máxima do cursor para a view atual
func (m Model) getMaxCursorPos() int {
	switch m.currentView {
	case ViewAlerts:
		return len(m.getFilteredAnomalies()) - 1
	case ViewClusters:
		return len(m.clusters) - 1
	default:
		return 0
	}
}

// getFilteredAnomalies retorna anomalias filtradas
func (m Model) getFilteredAnomalies() []analyzer.Anomaly {
	if m.filterSeverity == "All" && m.filterCluster == "" {
		return m.anomalies
	}

	filtered := []analyzer.Anomaly{}
	for _, a := range m.anomalies {
		// Filtro de severidade
		if m.filterSeverity != "All" && a.Severity.String() != m.filterSeverity {
			continue
		}

		// Filtro de cluster
		if m.filterCluster != "" && a.Cluster != m.filterCluster {
			continue
		}

		filtered = append(filtered, a)
	}

	return filtered
}

// getSortedClusters retorna clusters ordenados por nome
func (m Model) getSortedClusters() []*ClusterInfo {
	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, c := range m.clusters {
		clusters = append(clusters, c)
	}

	// TODO: Ordenar por nome ou outro critério
	return clusters
}

// Helper para criar chave
func makeKey(cluster, namespace, name string) string {
	return cluster + "/" + namespace + "/" + name
}

// GetSnapshotChan retorna canal de snapshots (para testes/integração)
func (m Model) GetSnapshotChan() chan *models.HPASnapshot {
	return m.snapshotChan
}

// GetAnomalyChan retorna canal de anomalias (para testes/integração)
func (m Model) GetAnomalyChan() chan analyzer.Anomaly {
	return m.anomalyChan
}

// Messages para Bubble Tea
type tickMsg time.Time
type snapshotMsg struct {
	snapshot *models.HPASnapshot
}
type anomalyMsg struct {
	anomaly analyzer.Anomaly
}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForSnapshot(ch chan *models.HPASnapshot) tea.Cmd {
	return func() tea.Msg {
		snapshot := <-ch
		return snapshotMsg{snapshot: snapshot}
	}
}

func waitForAnomaly(ch chan analyzer.Anomaly) tea.Cmd {
	return func() tea.Msg {
		anomaly := <-ch
		return anomalyMsg{anomaly: anomaly}
	}
}
