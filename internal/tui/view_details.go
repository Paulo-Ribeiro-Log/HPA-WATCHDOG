package tui

import (
	"fmt"
	"strings"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDetails() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("🔍 HPA Watchdog - Detalhes da Anomalia")
	content.WriteString(header + "\n\n")

	// Tabs
	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	anomalies := m.getFilteredAnomalies()

	if len(anomalies) == 0 || m.selectedAnomaly >= len(anomalies) {
		emptyMsg := BoxStyle.Copy().Width(m.width - 4).Render(
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Nenhuma anomalia selecionada"),
		)
		content.WriteString(emptyMsg + "\n\n")
		content.WriteString(HelpStyle.Render("Dica: Pressione Tab para voltar à view de Alertas"))
		content.WriteString("\n\n")
	} else {
		// Renderiza detalhes da anomalia
		anomaly := anomalies[m.selectedAnomaly]
		details := m.renderAnomalyDetails(anomaly)
		content.WriteString(details + "\n\n")
	}

	// Footer
	footer := m.renderFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderAnomalyDetails(a analyzer.Anomaly) string {
	var lines []string

	// Título
	title := SectionTitleStyle.Render(fmt.Sprintf("🔔 %s", string(a.Type)))
	severity := SeverityBadge(a.Severity.String())
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", severity))
	lines = append(lines, "")

	// Informações básicas
	lines = append(lines, TableHeaderStyle.Render("📍 Localização"))
	lines = append(lines, RenderMetric("Cluster:", a.Cluster))
	lines = append(lines, RenderMetric("Namespace:", a.Namespace))
	lines = append(lines, RenderMetric("HPA:", a.HPAName))
	lines = append(lines, RenderMetric("Timestamp:", a.Timestamp.Format("2006-01-02 15:04:05")))
	lines = append(lines, "")

	// Mensagem
	lines = append(lines, TableHeaderStyle.Render("📝 Descrição"))
	lines = append(lines, wrapText(a.Message, m.width-10))
	lines = append(lines, "")

	// Snapshot (se disponível)
	if a.Snapshot != nil {
		lines = append(lines, TableHeaderStyle.Render("📊 Métricas do HPA"))
		lines = append(lines, RenderMetric("Réplicas atuais:", fmt.Sprintf("%d", a.Snapshot.CurrentReplicas)))
		lines = append(lines, RenderMetric("Réplicas min/max:", fmt.Sprintf("%d / %d", a.Snapshot.MinReplicas, a.Snapshot.MaxReplicas)))

		if a.Snapshot.CPUCurrent > 0 {
			cpuStyle := MetricValueStyle
			if a.Snapshot.CPUCurrent > 80 {
				cpuStyle = StatusWarningStyle
			}
			if a.Snapshot.CPUCurrent > 90 {
				cpuStyle = StatusCriticalStyle
			}
			lines = append(lines, MetricLabelStyle.Render("CPU atual:")+" "+cpuStyle.Render(fmt.Sprintf("%.1f%%", a.Snapshot.CPUCurrent)))
		}

		if a.Snapshot.MemoryCurrent > 0 {
			memStyle := MetricValueStyle
			if a.Snapshot.MemoryCurrent > 80 {
				memStyle = StatusWarningStyle
			}
			if a.Snapshot.MemoryCurrent > 90 {
				memStyle = StatusCriticalStyle
			}
			lines = append(lines, MetricLabelStyle.Render("Memory atual:")+" "+memStyle.Render(fmt.Sprintf("%.1f%%", a.Snapshot.MemoryCurrent)))
		}

		if a.Snapshot.ErrorRate > 0 {
			errStyle := MetricValueStyle
			if a.Snapshot.ErrorRate > 5 {
				errStyle = StatusCriticalStyle
			}
			lines = append(lines, MetricLabelStyle.Render("Error rate:")+" "+errStyle.Render(fmt.Sprintf("%.2f%%", a.Snapshot.ErrorRate)))
		}

		if a.Snapshot.P95Latency > 0 {
			latStyle := MetricValueStyle
			if a.Snapshot.P95Latency > 1000 {
				latStyle = StatusWarningStyle
			}
			lines = append(lines, MetricLabelStyle.Render("P95 Latency:")+" "+latStyle.Render(fmt.Sprintf("%.0fms", a.Snapshot.P95Latency)))
		}

		lines = append(lines, "")
	}

	// Ações sugeridas
	if len(a.Actions) > 0 {
		lines = append(lines, TableHeaderStyle.Render("🔧 Ações Sugeridas"))
		for i, action := range a.Actions {
			bullet := StatusInfoStyle.Render(fmt.Sprintf("%d.", i+1))
			actionText := wrapText(action, m.width-15)
			lines = append(lines, bullet+" "+actionText)
		}
		lines = append(lines, "")
	}

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

// wrapText quebra texto em múltiplas linhas
func wrapText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}

	var result []string
	words := strings.Fields(text)
	var line string

	for _, word := range words {
		testLine := line
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > maxWidth {
			if line != "" {
				result = append(result, line)
				line = word
			} else {
				// Palavra muito longa, quebra mesmo assim
				result = append(result, word)
				line = ""
			}
		} else {
			line = testLine
		}
	}

	if line != "" {
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
