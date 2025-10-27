package tui

import (
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewType define os tipos de views disponíveis
type ViewType int

const (
	ViewSetup ViewType = iota // Setup inicial (NOVA VIEW)
	ViewDashboard
	ViewAlerts
	ViewClusters
	ViewHistory       // Análise histórica com gráficos
	ViewStressTest    // Dashboard de stress test (em tempo real)
	ViewStressReport  // Relatório final do stress test
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
	scanRunning   bool      // Indica se scan está em execução
	scanPaused    bool      // Indica se scan está pausado
	scanStartTime time.Time // Momento em que o scan foi iniciado

	// Stress Test (apenas em modo stress test)
	stressTestResult *models.StressTestMetrics // Resultado final do stress test

	// Canais de dados (recebe atualizações do monitor)
	snapshotChan chan *models.HPASnapshot
	anomalyChan  chan analyzer.Anomaly

	// Canais de controle
	setupDoneChan      chan struct{}
	pauseChan          chan struct{}
	stopChan           chan struct{}
	restartChan        chan struct{} // Canal para sinalizar reinício
	scanStatusChan     chan scanStatusMsg
	stressResultChan   chan *models.StressTestMetrics // Canal para receber resultado final do stress test
}

// ClusterInfo informações resumidas de um cluster
type ClusterInfo struct {
	Name              string
	TotalHPAs         int                 // Total de HPAs únicos descobertos
	TotalAnomalies    int
	TotalScans        int                 // Quantidade de ciclos de scan completos
	Status            string              // "Online", "Offline", "Error"
	LastScan          time.Time           // Último snapshot recebido
	lastKnownScanTime time.Time           // Timestamp do último ciclo de scan registrado (interno)
	hpasDiscovered    map[string]struct{} // Conjunto de HPAs únicos descobertos
}

// New cria nova instância do model
func New() Model {
	return Model{
		currentView:      ViewSetup, // Inicia com setup
		setupState:       NewSetupState(),
		snapshots:        make(map[string]*models.TimeSeriesData),
		anomalies:        []analyzer.Anomaly{},
		clusters:         make(map[string]*ClusterInfo),
		filterSeverity:   "All",
		filterCluster:    "",
		autoRefresh:      true,
		snapshotChan:     make(chan *models.HPASnapshot, 100),
		anomalyChan:      make(chan analyzer.Anomaly, 100),
		setupDoneChan:    make(chan struct{}, 1),
		pauseChan:        make(chan struct{}, 1),
		stopChan:         make(chan struct{}, 1),
		restartChan:      make(chan struct{}, 1), // Canal de reinício
		scanStatusChan:   make(chan scanStatusMsg, 10),
		stressResultChan: make(chan *models.StressTestMetrics, 1), // Canal de resultado do stress test
	}
}

// Init inicializa o model (Bubble Tea)
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForSnapshot(m.snapshotChan),
		waitForAnomaly(m.anomalyChan),
		waitForScanStatus(m.scanStatusChan),
		waitForStressResult(m.stressResultChan),
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

	case scanStatusMsg:
		m.scanRunning = msg.running
		m.scanPaused = msg.paused
		if !msg.startTime.IsZero() {
			m.scanStartTime = msg.startTime
		}
		return m, waitForScanStatus(m.scanStatusChan)

	case stressResultMsg:
		// Recebeu resultado final do stress test
		m.stressTestResult = msg.result
		// Muda automaticamente para a view de relatório
		m.currentView = ViewStressReport
		return m, waitForStressResult(m.stressResultChan)
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
	case ViewHistory:
		return m.renderHistory()
	case ViewStressTest:
		return m.renderStressTest()
	case ViewStressReport:
		return m.renderStressReport()
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
		// Navega entre as views principais (exceto Setup)
		// Ciclo: Dashboard -> Alertas -> Clusters -> Histórico -> Stress Test -> Detalhes -> Dashboard
		switch m.currentView {
		case ViewDashboard:
			m.currentView = ViewAlerts
		case ViewAlerts:
			m.currentView = ViewClusters
		case ViewClusters:
			m.currentView = ViewHistory
		case ViewHistory:
			m.currentView = ViewStressTest
		case ViewStressTest:
			m.currentView = ViewDetails
		case ViewDetails:
			m.currentView = ViewDashboard
		default:
			// Se estiver em qualquer outra view (ex: Setup), vai para Dashboard
			m.currentView = ViewDashboard
		}
		m.cursorPos = 0
		return m, nil

	case "shift+tab":
		// Navega para trás entre as views principais (exceto Setup)
		// Ciclo reverso: Dashboard -> Detalhes -> Stress Test -> Histórico -> Clusters -> Alertas -> Dashboard
		switch m.currentView {
		case ViewDashboard:
			m.currentView = ViewDetails
		case ViewAlerts:
			m.currentView = ViewDashboard
		case ViewClusters:
			m.currentView = ViewAlerts
		case ViewHistory:
			m.currentView = ViewClusters
		case ViewStressTest:
			m.currentView = ViewHistory
		case ViewDetails:
			m.currentView = ViewStressTest
		default:
			// Se estiver em qualquer outra view (ex: Setup), vai para Dashboard
			m.currentView = ViewDashboard
		}
		m.cursorPos = 0
		return m, nil

	case "h", "home":
		// Volta para a primeira view (Dashboard)
		m.currentView = ViewDashboard
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

	case "R":
		// Reiniciar aplicação (shift+r)
		return m.handleRestart()

	case "p":
		// Pausar/Retomar scan
		if m.scanRunning {
			// Sinaliza pausa via canal
			select {
			case m.pauseChan <- struct{}{}:
			default:
			}
		}
		return m, nil

	case "x":
		// Cancelar teste (para sem gerar relatório)
		if m.scanRunning {
			// Sinaliza cancelamento via canal de stop
			select {
			case m.stopChan <- struct{}{}:
			default:
			}
			// Limpa estado do teste
			m.scanRunning = false
			m.scanPaused = false
		}
		return m, nil

	case "s":
		// Parar teste e gerar relatório
		if m.scanRunning {
			// Sinaliza stop via canal
			select {
			case m.stopChan <- struct{}{}:
			default:
			}
			// TODO: Acionar geração de relatório
			// Por enquanto, apenas para o teste
			m.scanRunning = false
			m.scanPaused = false
		}
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
			Name:           snapshot.Cluster,
			Status:         "Online",
			hpasDiscovered: make(map[string]struct{}),
		}
		m.clusters[snapshot.Cluster] = cluster
	}

	// Rastreia HPAs únicos descobertos
	hpaKey := snapshot.Namespace + "/" + snapshot.Name
	cluster.hpasDiscovered[hpaKey] = struct{}{}
	cluster.TotalHPAs = len(cluster.hpasDiscovered)

	// Detecta novo ciclo de scan baseado em mudança significativa no timestamp
	// Se LastScan mudou mais de 10 segundos desde o último conhecido, é um novo ciclo
	currentTime := time.Now()
	if cluster.lastKnownScanTime.IsZero() {
		// Primeiro scan deste cluster
		cluster.TotalScans = 1
		cluster.lastKnownScanTime = currentTime
	} else if currentTime.Sub(cluster.lastKnownScanTime) > 10*time.Second {
		// Novo ciclo de scan detectado
		cluster.TotalScans++
		cluster.lastKnownScanTime = currentTime
	}

	cluster.LastScan = currentTime
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

// handleRestart reinicia a aplicação limpando dados e sinalizando reinício
func (m Model) handleRestart() (tea.Model, tea.Cmd) {
	// Salva a view atual para decidir comportamento
	wasInStressTest := m.currentView == ViewStressTest
	wasRunning := m.scanRunning

	// Para o scan se estiver rodando
	if m.scanRunning {
		select {
		case m.stopChan <- struct{}{}:
		default:
		}
	}

	// Aguarda um momento para o engine processar o stop
	time.Sleep(100 * time.Millisecond)

	// Limpa todos os dados em cache
	m.snapshots = make(map[string]*models.TimeSeriesData)
	m.anomalies = []analyzer.Anomaly{}
	m.clusters = make(map[string]*ClusterInfo)

	// Reseta estado
	m.scanRunning = false
	m.scanPaused = false
	m.scanStartTime = time.Time{}
	m.cursorPos = 0
	m.selectedCluster = ""
	m.selectedNamespace = ""
	m.selectedHPA = ""
	m.selectedAnomaly = 0
	m.filterSeverity = "All"
	m.filterCluster = ""

	// Se estava em stress test e rodando, mantém na view e sinaliza restart
	if wasInStressTest && wasRunning {
		m.currentView = ViewStressTest

		// Sinaliza reinício automático do teste
		if m.restartChan != nil {
			select {
			case m.restartChan <- struct{}{}:
			default:
			}
		}
	} else {
		// Caso contrário, volta para dashboard
		m.currentView = ViewDashboard
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
	case ViewStressTest:
		// Retorna número de HPAs disponíveis para navegação
		return len(m.snapshots) - 1
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

// getSortedClusters retorna clusters ordenados por nome alfabeticamente
func (m Model) getSortedClusters() []*ClusterInfo {
	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, c := range m.clusters {
		clusters = append(clusters, c)
	}

	// Ordenação alfabética por nome (bubble sort simples)
	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			if clusters[i].Name > clusters[j].Name {
				clusters[i], clusters[j] = clusters[j], clusters[i]
			}
		}
	}

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

// GetSetupDoneChan retorna canal de conclusão do setup
func (m Model) GetSetupDoneChan() chan struct{} {
	return m.setupDoneChan
}

// GetPauseChan retorna canal de pausa
func (m Model) GetPauseChan() chan struct{} {
	return m.pauseChan
}

// GetStopChan retorna canal de stop
func (m Model) GetStopChan() chan struct{} {
	return m.stopChan
}

// GetRestartChan retorna canal de restart
func (m Model) GetRestartChan() chan struct{} {
	return m.restartChan
}

// GetStressResultChan retorna canal de resultado do stress test
func (m Model) GetStressResultChan() chan *models.StressTestMetrics {
	return m.stressResultChan
}

// GetScanStatusChan retorna canal de status do scan
func (m Model) GetScanStatusChan() chan scanStatusMsg {
	return m.scanStatusChan
}

// UpdateScanStatus envia atualização de status do scan
func (m Model) UpdateScanStatus(running, paused bool, startTime time.Time) {
	select {
	case m.scanStatusChan <- scanStatusMsg{
		running:   running,
		paused:    paused,
		startTime: startTime,
	}:
	default:
		// Canal cheio, ignora
	}
}

// GetScanConfig retorna configuração do scan
func (m Model) GetScanConfig() *scanner.ScanConfig {
	if m.setupState == nil {
		return nil
	}
	return m.setupState.config
}

// SetScanRunning define estado de execução
func (m *Model) SetScanRunning(running bool) {
	m.scanRunning = running
}

// SetScanPaused define estado de pausa
func (m *Model) SetScanPaused(paused bool) {
	m.scanPaused = paused
}

// SetScanStartTime define o tempo de início do scan
func (m *Model) SetScanStartTime(startTime time.Time) {
	m.scanStartTime = startTime
}

// GetTimeRemaining calcula tempo restante do scan
func (m Model) GetTimeRemaining() time.Duration {
	if m.setupState == nil || m.setupState.config == nil {
		return 0
	}

	// Se duração é 0 (infinito), retorna 0
	if m.setupState.config.Duration == 0 {
		return 0
	}

	// Se scan não está rodando, retorna duração total
	if !m.scanRunning || m.scanStartTime.IsZero() {
		return m.setupState.config.Duration
	}

	// Calcula tempo decorrido
	elapsed := time.Since(m.scanStartTime)

	// Calcula tempo restante
	remaining := m.setupState.config.Duration - elapsed
	if remaining < 0 {
		return 0
	}

	return remaining
}

// GetScanProgress retorna progresso do scan em porcentagem (0-100)
func (m Model) GetScanProgress() float64 {
	if m.setupState == nil || m.setupState.config == nil {
		return 0
	}

	// Se duração é 0 (infinito), retorna 0
	if m.setupState.config.Duration == 0 {
		return 0
	}

	// Se scan não está rodando, retorna 0
	if !m.scanRunning || m.scanStartTime.IsZero() {
		return 0
	}

	// Calcula progresso
	elapsed := time.Since(m.scanStartTime)
	progress := (float64(elapsed) / float64(m.setupState.config.Duration)) * 100

	if progress > 100 {
		return 100
	}

	return progress
}

// Messages para Bubble Tea
type tickMsg time.Time
type snapshotMsg struct {
	snapshot *models.HPASnapshot
}
type anomalyMsg struct {
	anomaly analyzer.Anomaly
}
type scanStatusMsg struct {
	running   bool
	paused    bool
	startTime time.Time
}

type stressResultMsg struct {
	result *models.StressTestMetrics
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

func waitForScanStatus(ch chan scanStatusMsg) tea.Cmd {
	return func() tea.Msg {
		status := <-ch
		return status
	}
}

func waitForStressResult(ch chan *models.StressTestMetrics) tea.Cmd {
	return func() tea.Msg {
		result := <-ch
		return stressResultMsg{result: result}
	}
}
