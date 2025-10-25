package tui

import (
	"fmt"
	"strings"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderAlerts() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("🔔 HPA Watchdog - Alertas")
	content.WriteString(header + "\n\n")

	// Tabs
	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	// Filtros
	filters := m.renderFilters()
	content.WriteString(filters + "\n\n")

	// Lista de anomalias
	anomalies := m.getFilteredAnomalies()

	if len(anomalies) == 0 {
		emptyMsg := BoxStyle.Copy().Width(m.width - 4).Render(
			StatusOKStyle.Render("✓ Nenhuma anomalia detectada com os filtros atuais"),
		)
		content.WriteString(emptyMsg + "\n\n")
	} else {
		// Renderiza lista de anomalias
		anomalyList := m.renderAnomalyList(anomalies)
		content.WriteString(anomalyList + "\n\n")
	}

	// Footer
	footer := m.renderFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderFilters() string {
	// Filtro de severidade
	severities := []string{"All", "Critical", "Warning", "Info"}
	var severityButtons []string

	for i, sev := range severities {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(ColorTextSecondary)

		if sev == m.filterSeverity {
			style = style.
				Background(ColorBgSecondary).
				Foreground(ColorPrimary).
				Bold(true)
		}

		button := style.Render(fmt.Sprintf("[%d] %s", i+1, sev))
		severityButtons = append(severityButtons, button)
	}

	filterRow := lipgloss.JoinHorizontal(lipgloss.Top, severityButtons...)

	// Contador de anomalias
	totalStr := fmt.Sprintf("Total: %d anomalias", len(m.getFilteredAnomalies()))
	counter := MetricLabelStyle.Render(totalStr)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		SectionTitleStyle.Render("Filtros: "),
		" ",
		filterRow,
		"    ",
		counter,
	)
}

func (m Model) renderAnomalyList(anomalies []analyzer.Anomaly) string {
	var lines []string

	// Header da tabela
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableHeaderStyle.Copy().Width(10).Render("Hora"),
		TableHeaderStyle.Copy().Width(12).Render("Severidade"),
		TableHeaderStyle.Copy().Width(18).Render("Tipo"),
		TableHeaderStyle.Copy().Width(30).Render("HPA"),
		TableHeaderStyle.Copy().Width(m.width - 74).Render("Mensagem"),
	)
	lines = append(lines, headerLine)
	lines = append(lines, Divider(m.width-4))

	// Calcula quantas linhas cabem na tela
	availableHeight := m.height - 15 // Reserva espaço para header/footer
	maxLines := availableHeight
	if maxLines < 1 {
		maxLines = 10
	}
	if len(anomalies) < maxLines {
		maxLines = len(anomalies)
	}

	// Renderiza linhas
	for i := 0; i < maxLines; i++ {
		a := anomalies[i]

		// Estilo da linha (highlight se selecionada)
		rowStyle := TableRowStyle
		if i == m.cursorPos {
			rowStyle = TableRowSelectedStyle
		}

		timestamp := a.Timestamp.Format("15:04:05")
		severity := SeverityBadge(a.Severity.String())
		anomalyType := AnomalyTypeBadge(string(a.Type))
		hpa := fmt.Sprintf("%s/%s/%s", a.Cluster, a.Namespace, a.HPAName)
		message := truncate(a.Message, m.width-74)

		line := lipgloss.JoinHorizontal(lipgloss.Top,
			rowStyle.Copy().Width(10).Render(timestamp),
			rowStyle.Copy().Width(12).Render(severity),
			rowStyle.Copy().Width(18).Render(anomalyType),
			rowStyle.Copy().Width(30).Render(hpa),
			rowStyle.Copy().Width(m.width-74).Render(message),
		)

		lines = append(lines, line)
	}

	// Indicador de mais itens
	if len(anomalies) > maxLines {
		moreMsg := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(
			fmt.Sprintf("... e mais %d anomalias (use ↑↓ para navegar)", len(anomalies)-maxLines),
		)
		lines = append(lines, "", moreMsg)
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}
