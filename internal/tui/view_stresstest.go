package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// Timezone local (GMT-3 - Horário de Brasília)
var localTimezone = time.FixedZone("BRT", -3*60*60)

// toLocalTime converte um timestamp UTC para o timezone local
func toLocalTime(t time.Time) time.Time {
	return t.In(localTimezone)
}

// customTimeLabelFormatter cria um formatador de labels que força o uso do timezone local
func customTimeLabelFormatter() func(int, float64) string {
	return func(index int, value float64) string {
		// Converte o valor float64 (Unix timestamp) para time.Time
		t := time.Unix(int64(value), 0)
		// Converte para timezone local e formata
		return toLocalTime(t).Format("15:04:05")
	}
}

// Renderiza dashboard de stress test em tempo real com gráficos e menu de seleção
func (m Model) renderStressTest() string {
	var content strings.Builder

	// Header especial para stress test
	header := m.renderStressTestHeader()
	content.WriteString(header + "\n\n")

	// Se não há métricas de stress test ainda, mostra mensagem
	if m.setupState == nil || m.setupState.config == nil {
		emptyMsg := BoxStyle.Width(m.width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"",
				StatusInfoStyle.Render("🔥 Modo Stress Test não iniciado"),
				"",
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Configure um stress test no setup para começar."),
				"",
			),
		)
		content.WriteString(emptyMsg + "\n\n")

		footer := m.renderFooter()
		content.WriteString(footer)
		return content.String()
	}

	// Status box: Progresso do teste
	statusBox := m.renderStressTestStatus()
	content.WriteString(statusBox + "\n\n")

	// Layout principal: gráficos à esquerda + menu de HPAs à direita
	mainContent := m.renderStressTestMainContent()
	content.WriteString(mainContent + "\n\n")

	// Alertas críticos em destaque
	if len(m.anomalies) > 0 {
		criticalAlerts := m.renderStressTestAlerts()
		content.WriteString(criticalAlerts + "\n\n")
	}

	// Footer com controles específicos de stress test
	footer := m.renderStressTestFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderStressTestHeader() string {
	timestamp := toLocalTime(time.Now()).Format("15:04:05")

	// Status do teste
	var status string

	if m.scanRunning {
		if m.scanPaused {
			status = StatusWarningStyle.Render("⏸ PAUSADO")
		} else {
			status = StatusOKStyle.Render("🔥 RODANDO")
		}

		// Mostra progresso do teste
		if m.setupState != nil && m.setupState.config != nil {
			remaining := m.GetTimeRemaining()
			if remaining > 0 {
				progressValue := m.GetScanProgress()

				// Barra de progresso visual
				progressBar := renderProgressBar(int(progressValue), 30)

				hours := int(remaining.Hours())
				minutes := int(remaining.Minutes()) % 60
				seconds := int(remaining.Seconds()) % 60

				var timeStr string
				if hours > 0 {
					timeStr = fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
				} else if minutes > 0 {
					timeStr = fmt.Sprintf("%dm%ds", minutes, seconds)
				} else {
					timeStr = fmt.Sprintf("%ds", seconds)
				}

				status += fmt.Sprintf(" %s %.0f%% (%s restante)", progressBar, progressValue, timeStr)
			} else if m.setupState.config.Duration == 0 {
				status += lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(" | ∞ Infinito")
			}
		}
	} else {
		// Verifica se teste foi parado/finalizado
		if m.setupState.config != nil && m.getTotalScans() > 0 {
			status = StatusInfoStyle.Render("✓ FINALIZADO")
		} else {
			status = StatusInfoStyle.Render("○ PARADO")
		}
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("🔥 STRESS TEST MODE")

	right := lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(
		fmt.Sprintf("%s  %s", status, timestamp),
	)

	width := m.width
	if width < 1 {
		width = 80
	}

	gap := width - lipgloss.Width(title) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return title + strings.Repeat(" ", gap) + right
}

func renderProgressBar(percent int, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	filled := (percent * width) / 100
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	style := lipgloss.NewStyle().Foreground(ColorPrimary)
	if percent > 75 {
		style = style.Foreground(lipgloss.Color("#FFA500")) // Laranja
	}
	if percent > 90 {
		style = style.Foreground(lipgloss.Color("#FF0000")) // Vermelho
	}

	return style.Render(bar)
}

func (m Model) renderStressTestStatus() string {
	// Calcula métricas em tempo real
	totalHPAs := 0
	for _, cluster := range m.clusters {
		totalHPAs += cluster.TotalHPAs
	}

	totalAnomalies := len(m.anomalies)
	criticalCount := 0
	warningCount := 0

	for _, anomaly := range m.anomalies {
		switch anomaly.Severity.String() {
		case "Critical":
			criticalCount++
		case "Warning":
			warningCount++
		}
	}

	hpasWithIssues := m.countHPAsWithAnomalies()
	healthyHPAs := totalHPAs - hpasWithIssues
	healthPercent := 0.0
	if totalHPAs > 0 {
		healthPercent = float64(healthyHPAs) / float64(totalHPAs) * 100
	}

	// Determina status geral do teste
	var testStatus string
	var testStatusStyle lipgloss.Style
	if healthPercent >= 90 {
		testStatus = "✅ SAUDÁVEL"
		testStatusStyle = StatusOKStyle
	} else if healthPercent >= 70 {
		testStatus = "⚠️  ATENÇÃO"
		testStatusStyle = StatusWarningStyle
	} else {
		testStatus = "🔴 CRÍTICO"
		testStatusStyle = StatusCriticalStyle
	}

	// Linha 1: Status geral
	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		testStatusStyle.Render(testStatus),
		"  ",
		MetricLabelStyle.Render(fmt.Sprintf("Saúde: %.1f%% (%d/%d HPAs)", healthPercent, healthyHPAs, totalHPAs)),
	)

	// Linha 2: Distribuição de problemas
	line2 := ""
	if totalAnomalies > 0 {
		line2 = lipgloss.JoinHorizontal(lipgloss.Left,
			StatusCriticalStyle.Render(fmt.Sprintf("🔴 %d Críticos", criticalCount)),
			"  ",
			StatusWarningStyle.Render(fmt.Sprintf("⚠️  %d Avisos", warningCount)),
			"  ",
			StatusInfoStyle.Render(fmt.Sprintf("ℹ️  %d Info", totalAnomalies-criticalCount-warningCount)),
		)
	} else {
		line2 = StatusOKStyle.Render("✅ Nenhum problema detectado")
	}

	// Linha 3: Clusters monitorados
	line3 := MetricLabelStyle.Render(fmt.Sprintf("Clusters: %d  |  HPAs: %d  |  Scans: %d",
		len(m.clusters), totalHPAs, m.getTotalScans()))

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		line1,
		"",
		line2,
		"",
		line3,
		"",
	)

	return BoxStyle.Width(m.width - 4).Render(content)
}

// renderStressTestMainContent - Gráficos à esquerda, menu de HPAs à direita
func (m Model) renderStressTestMainContent() string {
	// Calcula larguras: 70% para gráficos, 30% para menu de HPAs
	graphsWidth := int(float64(m.width) * 0.65)
	menuWidth := m.width - graphsWidth - 6 // Deixa espaço para bordas

	if graphsWidth < 40 {
		graphsWidth = 40
	}
	if menuWidth < 25 {
		menuWidth = 25
	}

	// Painel esquerdo: gráficos de CPU e Memória lado a lado
	graphsPanel := m.renderMetricsGraphs(graphsWidth)

	// Painel direito: menu de seleção de HPAs
	hpaMenu := m.renderHPASelectionMenu(menuWidth)

	// Junta horizontalmente
	mainRow := lipgloss.JoinHorizontal(lipgloss.Top,
		graphsPanel,
		strings.Repeat(" ", 2),
		hpaMenu,
	)

	return mainRow
}

// renderMetricsGraphs - Gráficos de CPU e Memória do HPA selecionado
func (m Model) renderMetricsGraphs(width int) string {
	// Busca o HPA selecionado
	selectedKey := m.getSelectedHPAKey()
	ts, exists := m.snapshots[selectedKey]

	var cpuGraph, memoryGraph, replicasInfo string

	if !exists || ts == nil || len(ts.Snapshots) == 0 {
		// Sem dados para exibir
		emptyMsg := lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Render("Selecione um HPA no menu à direita →")

		cpuGraph = BoxStyle.Width(width).Height(12).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				SectionTitleStyle.Render("📊 CPU (%)"),
				"",
				emptyMsg,
			),
		)

		memoryGraph = BoxStyle.Width(width).Height(12).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				SectionTitleStyle.Render("💾 Memória (%)"),
				"",
				emptyMsg,
			),
		)
	} else {
		// Extrai dados históricos
		cpuData := make([]float64, 0, len(ts.Snapshots))
		memoryData := make([]float64, 0, len(ts.Snapshots))

		for _, snapshot := range ts.Snapshots {
			cpuData = append(cpuData, snapshot.CPUCurrent)
			memoryData = append(memoryData, snapshot.MemoryCurrent)
		}

		// Snapshot mais recente para estatísticas
		latest := ts.Snapshots[len(ts.Snapshots)-1]

		// Calcula estatísticas
		cpuStats := calculateStats(cpuData)
		memStats := calculateStats(memoryData)

		// Header do HPA selecionado com réplicas
		parts := strings.Split(selectedKey, "/")
		hpaShortName := parts[len(parts)-1]

		// Info de réplicas
		replicasInfo = m.renderReplicasInfo(&latest, width)

		// Renderiza gráfico de CPU com estatísticas
		cpuGraphContent := renderGraphWithSnapshots(ts.Snapshots, "cpu", width-4, 8)
		cpuStatsContent := m.renderStats(cpuStats, latest.CPUTarget, "%")

		cpuGraph = BoxStyle.Width(width).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				SectionTitleStyle.Render(fmt.Sprintf("📊 CPU - %s", hpaShortName)),
				"",
				cpuGraphContent,
				"",
				cpuStatsContent,
			),
		)

		// Renderiza gráfico de Memória com estatísticas
		memGraphContent := renderGraphWithSnapshots(ts.Snapshots, "memory", width-4, 8)
		memStatsContent := m.renderStats(memStats, latest.MemoryTarget, "%")

		memoryGraph = BoxStyle.Width(width).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				SectionTitleStyle.Render(fmt.Sprintf("💾 Memória - %s", hpaShortName)),
				"",
				memGraphContent,
				"",
				memStatsContent,
			),
		)
	}

	// Empilha verticalmente
	return lipgloss.JoinVertical(lipgloss.Left,
		replicasInfo,
		"",
		cpuGraph,
		"",
		memoryGraph,
	)
}

// renderGraphWithSnapshots - Renderiza um gráfico Time Series usando timestamps reais dos snapshots
func renderGraphWithSnapshots(snapshots []models.HPASnapshot, metricType string, width int, height int) string {
	if len(snapshots) == 0 {
		return lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Sem dados disponíveis")
	}

	// Limita dimensões mínimas
	if width < 40 {
		width = 40
	}
	if height < 8 {
		height = 8
	}

	// Extrai valores e timestamps reais dos snapshots
	timePoints := make([]timeserieslinechart.TimePoint, 0, len(snapshots))
	var minVal, maxVal float64

	for i, snapshot := range snapshots {
		var value float64
		switch metricType {
		case "cpu":
			value = snapshot.CPUCurrent
		case "memory":
			value = snapshot.MemoryCurrent
		default:
			value = snapshot.CPUCurrent
		}

		if i == 0 {
			minVal = value
			maxVal = value
		} else {
			if value < minVal {
				minVal = value
			}
			if value > maxVal {
				maxVal = value
			}
		}

		timePoints = append(timePoints, timeserieslinechart.TimePoint{
			Time:  toLocalTime(snapshot.Timestamp),  // Converte para timezone local (GMT-3)
			Value: value,
		})
	}

	// Adiciona margem para melhor visualização
	dataRange := maxVal - minVal
	if dataRange == 0 {
		dataRange = 1
	}
	minY := minVal - dataRange*0.1
	maxY := maxVal + dataRange*0.1

	// Cria formatador para eixo Y (valores com 1 decimal)
	yFormatter := func(index int, value float64) string {
		return fmt.Sprintf("%.1f", value)
	}

	// Cria time series chart com formatadores customizados
	tsc := timeserieslinechart.New(
		width, height,
		timeserieslinechart.WithYRange(minY, maxY),
		timeserieslinechart.WithYLabelFormatter(yFormatter),
		timeserieslinechart.WithXLabelFormatter(customTimeLabelFormatter()),
		timeserieslinechart.WithXYSteps(4, 5), // 4 labels no eixo X, 5 no eixo Y
	)

	// Adiciona todos os pontos
	for _, tp := range timePoints {
		tsc.Push(tp)
	}

	// Desenha o gráfico (necessário antes de View())
	tsc.Draw()

	// Renderiza
	return tsc.View()
}

// renderHPASelectionMenu - Menu interativo de seleção de HPAs com scroll
func (m Model) renderHPASelectionMenu(width int) string {
	var allLines []string

	// Header fixo
	allLines = append(allLines, SectionTitleStyle.Render("📋 HPAs Monitorados"))
	allLines = append(allLines, "")

	// Renderiza lista usando lista ordenada + cursorPos
	hpaList := m.getSortedHPAList()
	currentCluster := ""

	// Cria todas as linhas da lista (incluindo headers de cluster)
	var listItems []string
	for i, hpaKey := range hpaList {
		parts := strings.Split(hpaKey, "/")
		if len(parts) != 3 {
			continue
		}

		cluster := parts[0]
		hpaName := parts[1] + "/" + parts[2]

		// Header do cluster (quando muda)
		if cluster != currentCluster {
			if currentCluster != "" {
				listItems = append(listItems, "") // Espaço entre clusters
			}
			listItems = append(listItems, lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextSecondary).
				Render(fmt.Sprintf("▼ %s", cluster)))
			currentCluster = cluster
		}

		// Verifica se é o item selecionado usando índice + cursorPos
		isSelected := i == m.cursorPos

		// Verifica se HPA tem anomalias
		hasAnomaly := m.hpaHasAnomaly(hpaKey)

		// Estilo
		var itemStyle lipgloss.Style
		var prefix string

		if isSelected {
			prefix = "→ "
			itemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)
		} else {
			prefix = "  "
			itemStyle = lipgloss.NewStyle().Foreground(ColorTextPrimary)
		}

		// Indicador de status
		statusIcon := "✓"
		if hasAnomaly {
			statusIcon = "⚠️"
			if !isSelected {
				itemStyle = itemStyle.Foreground(lipgloss.Color("#FFA500"))
			}
		}

		item := itemStyle.Render(fmt.Sprintf("%s%s %s", prefix, statusIcon, truncate(hpaName, width-6)))
		listItems = append(listItems, item)
	}

	itemCount := len(hpaList)

	if itemCount == 0 {
		allLines = append(allLines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Nenhum HPA detectado"))
		allLines = append(allLines, "")
		allLines = append(allLines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Aguardando dados..."))

		content := lipgloss.JoinVertical(lipgloss.Left, allLines...)
		menuHeight := m.height / 2
		if menuHeight < 20 {
			menuHeight = 20
		}
		return BoxStyle.Width(width).Height(menuHeight).Render(content)
	}

	// Calcula viewport e scroll
	menuHeight := m.height / 2
	if menuHeight < 20 {
		menuHeight = 20
	}

	// Reserva espaço para header (2 linhas) e footer (2 linhas)
	viewportHeight := menuHeight - 4
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Calcula offset de scroll para manter o item selecionado visível
	scrollOffset := 0
	if len(listItems) > viewportHeight {
		// Calcula a posição do item selecionado na lista (incluindo headers de cluster)
		selectedLineIndex := 0
		itemsSeen := 0
		for idx, line := range listItems {
			// Pula linhas vazias e headers de cluster
			if strings.HasPrefix(line, "▼") || line == "" {
				if itemsSeen <= m.cursorPos {
					selectedLineIndex = idx + 1
				}
			} else {
				if itemsSeen == m.cursorPos {
					selectedLineIndex = idx
					break
				}
				itemsSeen++
			}
		}

		// Centraliza o item selecionado no viewport
		scrollOffset = selectedLineIndex - (viewportHeight / 2)
		if scrollOffset < 0 {
			scrollOffset = 0
		}
		if scrollOffset > len(listItems)-viewportHeight {
			scrollOffset = len(listItems) - viewportHeight
		}
	}

	// Extrai itens visíveis
	visibleItems := listItems
	hasMoreAbove := false
	hasMoreBelow := false

	if len(listItems) > viewportHeight {
		end := scrollOffset + viewportHeight
		if end > len(listItems) {
			end = len(listItems)
		}
		visibleItems = listItems[scrollOffset:end]
		hasMoreAbove = scrollOffset > 0
		hasMoreBelow = end < len(listItems)
	}

	// Indicador de scroll acima
	if hasMoreAbove {
		allLines = append(allLines, lipgloss.NewStyle().
			Foreground(ColorTextSecondary).
			Render("  ↑ mais itens acima..."))
	}

	// Adiciona itens visíveis
	allLines = append(allLines, visibleItems...)

	// Indicador de scroll abaixo
	if hasMoreBelow {
		allLines = append(allLines, lipgloss.NewStyle().
			Foreground(ColorTextSecondary).
			Render("  ↓ mais itens abaixo..."))
	}

	// Footer
	allLines = append(allLines, "")
	allLines = append(allLines, lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Render(fmt.Sprintf("↑↓: Navegar | %d/%d HPAs", m.cursorPos+1, itemCount)))

	content := lipgloss.JoinVertical(lipgloss.Left, allLines...)
	return BoxStyle.Width(width).Height(menuHeight).Render(content)
}

// getSelectedHPAKey retorna a chave do HPA selecionado usando cursorPos
func (m Model) getSelectedHPAKey() string {
	hpaList := m.getSortedHPAList()
	if len(hpaList) == 0 {
		return ""
	}

	// Usa cursorPos para selecionar o HPA (limitado ao tamanho da lista)
	index := m.cursorPos
	if index >= len(hpaList) {
		index = len(hpaList) - 1
	}
	if index < 0 {
		index = 0
	}

	return hpaList[index]
}

// getSortedHPAList retorna lista ordenada de HPAs para navegação consistente
func (m Model) getSortedHPAList() []string {
	hpaList := make([]string, 0, len(m.snapshots))
	for key := range m.snapshots {
		hpaList = append(hpaList, key)
	}
	sort.Strings(hpaList) // Ordena alfabeticamente para ordem consistente
	return hpaList
}

// hpaHasAnomaly verifica se um HPA tem anomalias
func (m Model) hpaHasAnomaly(hpaKey string) bool {
	parts := strings.Split(hpaKey, "/")
	if len(parts) != 3 {
		return false
	}

	cluster := parts[0]
	namespace := parts[1]
	hpa := parts[2]

	for _, anomaly := range m.anomalies {
		if anomaly.Cluster == cluster && anomaly.Namespace == namespace && anomaly.HPAName == hpa {
			return true
		}
	}

	return false
}

func (m Model) renderStressTestAlerts() string {
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("⚠️  ALERTAS ATIVOS"))
	lines = append(lines, "")

	// Filtra apenas alertas críticos e warnings para destaque
	alertCount := 0
	maxAlerts := 5 // Mostra até 5 alertas mais recentes

	for _, anomaly := range m.anomalies {
		if alertCount >= maxAlerts {
			break
		}

		if anomaly.Severity == models.SeverityCritical || anomaly.Severity == models.SeverityWarning {
			timestamp := anomaly.Timestamp.Format("15:04:05")
			severity := SeverityBadge(anomaly.Severity.String())
			anomalyType := AnomalyTypeBadge(string(anomaly.Type))
			hpa := lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(
				fmt.Sprintf("%s/%s/%s", anomaly.Cluster, anomaly.Namespace, anomaly.HPAName),
			)

			lines = append(lines, fmt.Sprintf("[%s] %s %s  %s",
				timestamp, severity, anomalyType, hpa))
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(ColorTextMuted).Render(truncate(anomaly.Message, m.width-10)))

			if alertCount < maxAlerts-1 {
				lines = append(lines, "")
			}

			alertCount++
		}
	}

	if alertCount == 0 {
		lines = append(lines, StatusOKStyle.Render("✅ Nenhum alerta crítico"))
	}

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStressTestFooter() string {
	help := "Tab: Mudar view  •  H/Home: Dashboard  •  ↑↓: Selecionar HPA  •  R/F5: Refresh  •  Shift+R: Reiniciar"

	// Controles específicos de stress test
	if m.scanRunning {
		if m.scanPaused {
			help += "  •  P: Retomar"
		} else {
			help += "  •  P: Pausar"
		}
		help += "  •  X: Parar teste  •  S: Parar e salvar relatório"
	} else {
		// Se teste foi finalizado, mostra mensagem diferente
		if m.getTotalScans() > 0 {
			help = "✓ Teste finalizado  •  Enter: Novo teste  •  Tab: Ver resultados  •  Q: Sair"
		} else {
			help += "  •  Enter: Iniciar novo stress test"
		}
	}

	if m.scanRunning {
		help += "  •  Q: Sair"
	}

	return FooterStyle.Width(m.width).Render(help)
}

// Helper: conta total de scans realizados
func (m Model) getTotalScans() int {
	total := 0
	for _, cluster := range m.clusters {
		total += cluster.TotalScans
	}
	return total
}

// MetricStats estatísticas de uma métrica
type MetricStats struct {
	Current float64
	Min     float64
	Max     float64
	Avg     float64
}

// calculateStats calcula estatísticas de um dataset
func calculateStats(data []float64) MetricStats {
	if len(data) == 0 {
		return MetricStats{}
	}

	stats := MetricStats{
		Current: data[len(data)-1],
		Min:     data[0],
		Max:     data[0],
		Avg:     0,
	}

	sum := 0.0
	for _, val := range data {
		if val < stats.Min {
			stats.Min = val
		}
		if val > stats.Max {
			stats.Max = val
		}
		sum += val
	}

	stats.Avg = sum / float64(len(data))
	return stats
}

// renderStats renderiza painel de estatísticas
func (m Model) renderStats(stats MetricStats, target int32, unit string) string {
	currentStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorTextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(ColorTextPrimary)

	// Determina se está acima do target
	var statusIcon string
	if target > 0 && stats.Current > float64(target) {
		statusIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("⚠ Acima do target")
	} else if target > 0 && stats.Current < float64(target)*0.8 {
		statusIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("✓ Abaixo do target")
	} else {
		statusIcon = lipgloss.NewStyle().Foreground(ColorTextMuted).Render("○ Normal")
	}

	lines := []string{
		currentStyle.Render(fmt.Sprintf("Atual: %.1f%s", stats.Current, unit)),
		valueStyle.Render(fmt.Sprintf("Min: %.1f%s  Max: %.1f%s  Média: %.1f%s",
			stats.Min, unit, stats.Max, unit, stats.Avg, unit)),
	}

	if target > 0 {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Target: %d%s", target, unit))+" "+statusIcon)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderReplicasInfo renderiza informação de réplicas
func (m Model) renderReplicasInfo(snapshot *models.HPASnapshot, width int) string {
	currentStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorTextSecondary)

	// Calcula progresso visual
	current := float64(snapshot.CurrentReplicas)
	min := float64(snapshot.MinReplicas)
	max := float64(snapshot.MaxReplicas)

	var progressBar string
	if max > min {
		percentage := (current - min) / (max - min)
		barWidth := 30
		filledWidth := int(percentage * float64(barWidth))
		if filledWidth > barWidth {
			filledWidth = barWidth
		}
		if filledWidth < 0 {
			filledWidth = 0
		}

		filled := strings.Repeat("█", filledWidth)
		empty := strings.Repeat("░", barWidth-filledWidth)
		progressBar = lipgloss.NewStyle().Foreground(ColorPrimary).Render(filled) +
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render(empty)
	}

	// Status das réplicas
	var statusIcon, statusText string
	if snapshot.CurrentReplicas >= snapshot.MaxReplicas {
		statusIcon = "🔴"
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("NO LIMITE")
	} else if snapshot.CurrentReplicas == snapshot.DesiredReplicas {
		statusIcon = "✅"
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("ESTÁVEL")
	} else {
		statusIcon = "⚡"
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("ESCALANDO")
	}

	lines := []string{
		SectionTitleStyle.Render("🔢 RÉPLICAS"),
		"",
		currentStyle.Render(fmt.Sprintf("Atual: %d", snapshot.CurrentReplicas)) +
			labelStyle.Render(fmt.Sprintf("  (Desejadas: %d)", snapshot.DesiredReplicas)),
		labelStyle.Render(fmt.Sprintf("Limites: %d - %d", snapshot.MinReplicas, snapshot.MaxReplicas)),
		"",
		progressBar,
		"",
		statusIcon + " " + statusText,
	}

	return BoxStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}
