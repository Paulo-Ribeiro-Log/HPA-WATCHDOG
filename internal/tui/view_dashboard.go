package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("📊 HPA Watchdog - Dashboard")
	content.WriteString(header + "\n\n")

	// Tabs
	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	// Se não houver dados ainda, mostra mensagem de aguardo
	if len(m.clusters) == 0 && len(m.snapshots) == 0 {
		emptyMsg := BoxStyle.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"",
				StatusInfoStyle.Render("⏳ Aguardando primeiros dados..."),
				"",
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("O scan foi iniciado. Os primeiros dados aparecerão em breve."),
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Isso pode levar alguns segundos dependendo do cluster."),
				"",
			),
		)
		content.WriteString(emptyMsg + "\n\n")

		// Footer
		footer := m.renderFooter()
		content.WriteString(footer)
		return content.String()
	}

	// Métricas gerais (3 colunas)
	metrics := m.renderMetricsRow()
	content.WriteString(metrics + "\n\n")

	// Resumo de anomalias por severidade
	anomalySummary := m.renderAnomalySummary()
	content.WriteString(anomalySummary + "\n\n")

	// Top 5 clusters com mais anomalias
	topClusters := m.renderTopClusters(5)
	content.WriteString(topClusters + "\n\n")

	// Anomalias recentes (últimas 5)
	recentAnomalies := m.renderRecentAnomalies(5)
	content.WriteString(recentAnomalies + "\n\n")

	// Footer com help
	footer := m.renderFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderHeader(title string) string {
	timestamp := time.Now().Format("15:04:05")

	// Status do scan
	var status string
	if m.scanRunning {
		if m.scanPaused {
			status = StatusWarningStyle.Render("⏸ PAUSADO")
		} else {
			status = StatusOKStyle.Render("● RODANDO")
		}

		// Adiciona tempo restante se configurado
		if m.setupState != nil && m.setupState.config != nil {
			remaining := m.GetTimeRemaining()
			if remaining > 0 {
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

				progress := m.GetScanProgress()
				status += lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(
					fmt.Sprintf(" | Restante: %s (%.0f%%)", timeStr, progress),
				)
			} else if m.setupState.config.Duration == 0 {
				status += lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(" | ∞ Infinito")
			}
		}
	} else {
		status = StatusInfoStyle.Render("○ PARADO")
	}

	left := HeaderStyle.Render(title)
	right := lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(
		fmt.Sprintf("%s %s", status, timestamp),
	)

	width := m.width
	if width < 1 {
		width = 80
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderTabs() string {
	tabs := []string{
		"Dashboard",
		"Alertas",
		"Clusters",
		"Detalhes",
	}

	// Mapeia corretamente tab index para ViewType (pula ViewSetup que é índice 0)
	viewMapping := []ViewType{
		ViewDashboard, // Tab 0
		ViewAlerts,    // Tab 1
		ViewClusters,  // Tab 2
		ViewDetails,   // Tab 3
	}

	var rendered []string
	for i, tab := range tabs {
		style := TabStyle
		if viewMapping[i] == m.currentView {
			style = TabActiveStyle
		}
		rendered = append(rendered, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m Model) renderMetricsRow() string {
	// Calcula métricas
	totalClusters := len(m.clusters)
	totalHPAs := 0
	totalAnomalies := len(m.anomalies)
	criticalCount := 0
	warningCount := 0

	for _, cluster := range m.clusters {
		totalHPAs += cluster.TotalHPAs
	}

	for _, anomaly := range m.anomalies {
		switch anomaly.Severity.String() {
		case "Critical":
			criticalCount++
		case "Warning":
			warningCount++
		}
	}

	// Box 1: Clusters
	box1 := BoxStyle.Copy().Width(25).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			SectionTitleStyle.Render("Clusters"),
			"",
			RenderMetric("Total:", fmt.Sprintf("%d", totalClusters)),
			RenderMetric("Online:", fmt.Sprintf("%d", len(m.clusters))),
		),
	)

	// Box 2: HPAs
	box2 := BoxStyle.Copy().Width(25).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			SectionTitleStyle.Render("HPAs"),
			"",
			RenderMetric("Monitorados:", fmt.Sprintf("%d", totalHPAs)),
			RenderMetric("Com anomalias:", fmt.Sprintf("%d", m.countHPAsWithAnomalies())),
		),
	)

	// Box 3: Anomalias
	box3 := BoxStyle.Copy().Width(30).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			SectionTitleStyle.Render("Anomalias"),
			"",
			RenderMetric("Total:", fmt.Sprintf("%d", totalAnomalies)),
			StatusCriticalStyle.Render(fmt.Sprintf("Critical: %d", criticalCount))+" "+
				StatusWarningStyle.Render(fmt.Sprintf("Warning: %d", warningCount)),
		),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, box1, "  ", box2, "  ", box3)
}

func (m Model) renderAnomalySummary() string {
	// Conta anomalias por tipo
	typeCounts := make(map[string]int)
	for _, a := range m.anomalies {
		typeCounts[string(a.Type)]++
	}

	if len(typeCounts) == 0 {
		return BoxStyle.Copy().Width(m.width - 4).Render(
			SectionTitleStyle.Render("📈 Anomalias por Tipo") + "\n\n" +
				StatusOKStyle.Render("✓ Nenhuma anomalia detectada"),
		)
	}

	var lines []string
	lines = append(lines, SectionTitleStyle.Render("📈 Anomalias por Tipo"))
	lines = append(lines, "")

	// Ordena por contagem (top 5)
	var topTypes []struct {
		Type  string
		Count int
	}
	for t, c := range typeCounts {
		topTypes = append(topTypes, struct {
			Type  string
			Count int
		}{t, c})
	}

	// Simples ordenação decrescente
	for i := 0; i < len(topTypes); i++ {
		for j := i + 1; j < len(topTypes); j++ {
			if topTypes[j].Count > topTypes[i].Count {
				topTypes[i], topTypes[j] = topTypes[j], topTypes[i]
			}
		}
	}

	// Mostra top 5
	max := 5
	if len(topTypes) < max {
		max = len(topTypes)
	}

	for i := 0; i < max; i++ {
		badge := AnomalyTypeBadge(topTypes[i].Type)
		count := MetricValueStyle.Render(fmt.Sprintf("%d", topTypes[i].Count))
		lines = append(lines, fmt.Sprintf("%s  %s", badge, count))
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderTopClusters(limit int) string {
	clusters := m.getSortedClusters()

	if len(clusters) == 0 {
		return BoxStyle.Copy().Width(m.width - 4).Render(
			SectionTitleStyle.Render("🏢 Top Clusters") + "\n\n" +
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Nenhum cluster conectado"),
		)
	}

	var lines []string
	lines = append(lines, SectionTitleStyle.Render("🏢 Top Clusters (por anomalias)"))
	lines = append(lines, "")

	// Mostra até 'limit' clusters
	max := limit
	if len(clusters) < max {
		max = len(clusters)
	}

	for i := 0; i < max; i++ {
		c := clusters[i]
		status := ClusterStatusBadge(c.Status)
		name := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(c.Name)
		hpas := MetricLabelStyle.Render(fmt.Sprintf("HPAs: %d", c.TotalHPAs))
		anomalies := ""
		if c.TotalAnomalies > 0 {
			anomalies = StatusWarningStyle.Render(fmt.Sprintf("  Anomalias: %d", c.TotalAnomalies))
		} else {
			anomalies = StatusOKStyle.Render("  ✓ OK")
		}

		lines = append(lines, fmt.Sprintf("%s %s  %s%s", status, name, hpas, anomalies))
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderRecentAnomalies(limit int) string {
	anomalies := m.anomalies

	if len(anomalies) == 0 {
		return BoxStyle.Copy().Width(m.width - 4).Render(
			SectionTitleStyle.Render("🔔 Anomalias Recentes") + "\n\n" +
				StatusOKStyle.Render("✓ Nenhuma anomalia recente"),
		)
	}

	var lines []string
	lines = append(lines, SectionTitleStyle.Render("🔔 Anomalias Recentes"))
	lines = append(lines, "")

	// Mostra até 'limit' anomalias
	max := limit
	if len(anomalies) < max {
		max = len(anomalies)
	}

	for i := 0; i < max; i++ {
		a := anomalies[i]
		timestamp := a.Timestamp.Format("15:04:05")
		severity := SeverityBadge(a.Severity.String())
		anomalyType := AnomalyTypeBadge(string(a.Type))
		hpa := lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(
			fmt.Sprintf("%s/%s/%s", a.Cluster, a.Namespace, a.HPAName),
		)

		lines = append(lines, fmt.Sprintf("[%s] %s %s  %s",
			timestamp, severity, anomalyType, hpa))
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(ColorTextMuted).Render(truncate(a.Message, m.width-10)))
		if i < max-1 {
			lines = append(lines, "")
		}
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderFooter() string {
	help := "Tab: Mudar view  •  H/Home: Primeira view  •  ↑↓/jk: Navegar  •  Enter: Selecionar  •  1-4: Filtros  •  R/F5: Refresh"

	// Adiciona status de scan e tecla P se scan estiver rodando
	if m.scanRunning {
		if m.scanPaused {
			help += "  •  P: Retomar scan"
		} else {
			help += "  •  P: Pausar scan"
		}
	}

	help += "  •  Q: Sair"

	return FooterStyle.Copy().Width(m.width).Render(help)
}

// Helper functions
func (m Model) countHPAsWithAnomalies() int {
	hpas := make(map[string]bool)
	for _, a := range m.anomalies {
		key := makeKey(a.Cluster, a.Namespace, a.HPAName)
		hpas[key] = true
	}
	return len(hpas)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
