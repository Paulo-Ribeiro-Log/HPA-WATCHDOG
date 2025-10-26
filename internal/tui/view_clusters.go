package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderClusters() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("🏢 HPA Watchdog - Clusters")
	content.WriteString(header + "\n\n")

	// Tabs
	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	clusters := m.getSortedClusters()

	if len(clusters) == 0 {
		emptyMsg := BoxStyle.Copy().Width(m.width - 4).Render(
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Nenhum cluster conectado"),
		)
		content.WriteString(emptyMsg + "\n\n")
	} else {
		// Renderiza lista de clusters
		clusterList := m.renderClusterList(clusters)
		content.WriteString(clusterList + "\n\n")
	}

	// Footer
	footer := m.renderFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderClusterList(clusters []*ClusterInfo) string {
	var lines []string

	// Header da tabela
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableHeaderStyle.Copy().Width(15).Render("Status"),
		TableHeaderStyle.Copy().Width(30).Render("Cluster"),
		TableHeaderStyle.Copy().Width(10).Render("HPAs"),
		TableHeaderStyle.Copy().Width(12).Render("Anomalias"),
		TableHeaderStyle.Copy().Width(18).Render("Último Scan"),
		TableHeaderStyle.Copy().Width(20).Render("Tempo Restante"),
	)
	lines = append(lines, headerLine)
	lines = append(lines, Divider(m.width-4))

	// Renderiza clusters
	for i, cluster := range clusters {
		// Estilo da linha
		rowStyle := TableRowStyle
		if i == m.cursorPos {
			rowStyle = TableRowSelectedStyle
		}

		status := ClusterStatusBadge(cluster.Status)
		name := cluster.Name
		hpas := fmt.Sprintf("%d", cluster.TotalHPAs)
		anomalies := ""
		if cluster.TotalAnomalies > 0 {
			anomalies = StatusWarningStyle.Render(fmt.Sprintf("%d", cluster.TotalAnomalies))
		} else {
			anomalies = StatusOKStyle.Render("0")
		}

		lastScan := "-"
		if !cluster.LastScan.IsZero() {
			lastScan = cluster.LastScan.Format("15:04:05")
		}

		// Tempo restante
		timeRemaining := "-"
		if m.scanRunning {
			remaining := m.GetTimeRemaining()
			if remaining > 0 {
				// Formata tempo restante
				hours := int(remaining.Hours())
				minutes := int(remaining.Minutes()) % 60
				seconds := int(remaining.Seconds()) % 60

				if hours > 0 {
					timeRemaining = fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
				} else if minutes > 0 {
					timeRemaining = fmt.Sprintf("%dm%ds", minutes, seconds)
				} else {
					timeRemaining = fmt.Sprintf("%ds", seconds)
				}
			} else if m.setupState != nil && m.setupState.config != nil && m.setupState.config.Duration == 0 {
				timeRemaining = "∞ Infinito"
			} else {
				timeRemaining = "Concluído"
			}
		}

		line := lipgloss.JoinHorizontal(lipgloss.Top,
			rowStyle.Copy().Width(15).Render(status),
			rowStyle.Copy().Width(30).Render(name),
			rowStyle.Copy().Width(10).Render(hpas),
			rowStyle.Copy().Width(12).Render(anomalies),
			rowStyle.Copy().Width(18).Render(lastScan),
			rowStyle.Copy().Width(20).Render(timeRemaining),
		)

		lines = append(lines, line)
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}
