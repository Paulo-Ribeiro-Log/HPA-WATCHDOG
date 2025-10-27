package tui

import (
	"fmt"
	"strings"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderStressReport() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("📊 HPA Watchdog - Relatório do Stress Test")
	content.WriteString(header + "\n\n")

	// Se não houver resultado ainda
	if m.stressTestResult == nil {
		emptyMsg := BoxStyle.Width(m.width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"",
				StatusInfoStyle.Render("⏳ Aguardando conclusão do stress test..."),
				"",
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("O relatório será gerado automaticamente ao final do teste."),
				"",
			),
		)
		content.WriteString(emptyMsg + "\n\n")

		// Footer
		footer := m.renderStressReportFooter()
		content.WriteString(footer)
		return content.String()
	}

	result := m.stressTestResult

	// Badge de resultado (PASS/FAIL)
	resultBadge := m.renderTestResultBadge(result)
	content.WriteString(resultBadge + "\n\n")

	// Resumo executivo
	summary := m.renderExecutiveSummary(result)
	content.WriteString(summary + "\n\n")

	// Métricas de pico
	peakMetrics := m.renderPeakMetrics(result)
	content.WriteString(peakMetrics + "\n\n")

	// HPAs com problemas
	if len(result.CriticalIssues) > 0 || len(result.WarningIssues) > 0 {
		issues := m.renderTestIssues(result)
		content.WriteString(issues + "\n\n")
	}

	// Recomendações (se houver)
	if len(result.Recommendations) > 0 {
		recommendations := m.renderRecommendations(result)
		content.WriteString(recommendations + "\n\n")
	}

	// Footer com opções
	footer := m.renderStressReportFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderTestResultBadge(result *models.StressTestMetrics) string {
	testResult := result.GetTestResult()
	healthPct := result.GetHealthPercentage()

	var badge string
	var statusStyle lipgloss.Style

	if testResult == "PASS" {
		statusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#006600")).
			Padding(0, 2)
		badge = statusStyle.Render("✓ TESTE APROVADO")
	} else {
		statusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#CC0000")).
			Padding(0, 2)
		badge = statusStyle.Render("✗ TESTE REPROVADO")
	}

	healthBar := m.renderHealthBar(healthPct)

	info := lipgloss.JoinVertical(lipgloss.Left,
		badge,
		"",
		fmt.Sprintf("Saúde Geral: %.1f%%", healthPct),
		healthBar,
	)

	return BoxStyle.Width(m.width - 4).Render(info)
}

func (m Model) renderHealthBar(percentage float64) string {
	barWidth := 50
	filled := int(percentage * float64(barWidth) / 100)
	empty := barWidth - filled

	var color lipgloss.Color
	if percentage >= 90 {
		color = lipgloss.Color("#00FF00")
	} else if percentage >= 70 {
		color = lipgloss.Color("#FFFF00")
	} else {
		color = lipgloss.Color("#FF0000")
	}

	filledBar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	emptyBar := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(strings.Repeat("░", empty))

	return "[" + filledBar + emptyBar + "]"
}

func (m Model) renderExecutiveSummary(result *models.StressTestMetrics) string {
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("📋 Resumo Executivo"))
	lines = append(lines, "")

	// Informações básicas
	lines = append(lines,
		MetricLabelStyle.Render("Nome do Teste: ")+MetricValueStyle.Render(result.TestName),
		MetricLabelStyle.Render("Duração: ")+MetricValueStyle.Render(result.Duration.String()),
		MetricLabelStyle.Render("Status: ")+StatusStyle(result.Status),
		MetricLabelStyle.Render("Interval de Scan: ")+MetricValueStyle.Render(result.ScanInterval.String()),
		MetricLabelStyle.Render("Total de Scans: ")+MetricValueStyle.Render(fmt.Sprintf("%d", result.TotalScans)),
		"",
	)

	// Métricas de cobertura
	lines = append(lines,
		MetricLabelStyle.Render("Clusters Monitorados: ")+MetricValueStyle.Render(fmt.Sprintf("%d", result.TotalClusters)),
		MetricLabelStyle.Render("HPAs Monitorados: ")+MetricValueStyle.Render(fmt.Sprintf("%d", result.TotalHPAsMonitored)),
		MetricLabelStyle.Render("HPAs com Problemas: ")+StatusWarningStyle.Render(fmt.Sprintf("%d", result.TotalHPAsWithIssues)),
	)

	// Estatísticas de problemas
	if len(result.CriticalIssues) > 0 || len(result.WarningIssues) > 0 {
		lines = append(lines, "")
		lines = append(lines,
			StatusCriticalStyle.Render(fmt.Sprintf("Critical: %d", len(result.CriticalIssues)))+" | "+
				StatusWarningStyle.Render(fmt.Sprintf("Warnings: %d", len(result.WarningIssues)))+" | "+
				StatusInfoStyle.Render(fmt.Sprintf("Info: %d", len(result.InfoIssues))),
		)
	}

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderPeakMetrics(result *models.StressTestMetrics) string {
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("⚡ Métricas de Pico"))
	lines = append(lines, "")

	peaks := result.PeakMetrics

	// CPU
	if peaks.MaxCPUPercent > 0 {
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Render("CPU Máximo:"),
			fmt.Sprintf("  Valor: %s", MetricValueStyle.Render(fmt.Sprintf("%.1f%%", peaks.MaxCPUPercent))),
			fmt.Sprintf("  HPA: %s", lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(peaks.MaxCPUHPA)),
			fmt.Sprintf("  Horário: %s", toLocalTime(peaks.MaxCPUTime).Format("15:04:05")),
			"",
		)
	}

	// Memory
	if peaks.MaxMemoryPercent > 0 {
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Render("Memory Máximo:"),
			fmt.Sprintf("  Valor: %s", MetricValueStyle.Render(fmt.Sprintf("%.1f%%", peaks.MaxMemoryPercent))),
			fmt.Sprintf("  HPA: %s", lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(peaks.MaxMemoryHPA)),
			fmt.Sprintf("  Horário: %s", toLocalTime(peaks.MaxMemoryTime).Format("15:04:05")),
			"",
		)
	}

	// Réplicas
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Render("Evolução de Réplicas:"),
		fmt.Sprintf("  PRE (baseline):  %s réplicas", MetricValueStyle.Render(fmt.Sprintf("%d", peaks.TotalReplicasPre))),
		fmt.Sprintf("  PEAK (máximo):   %s réplicas", StatusWarningStyle.Render(fmt.Sprintf("%d", peaks.TotalReplicasPeak))),
		fmt.Sprintf("  POST (final):    %s réplicas", MetricValueStyle.Render(fmt.Sprintf("%d", peaks.TotalReplicasPost))),
		"",
		fmt.Sprintf("  Aumento: %s réplicas (%s)",
			StatusWarningStyle.Render(fmt.Sprintf("+%d", peaks.ReplicaIncrease)),
			StatusWarningStyle.Render(fmt.Sprintf("+%.1f%%", peaks.ReplicaIncreaseP)),
		),
	)

	// Error Rate
	if peaks.MaxErrorRate > 0 {
		lines = append(lines, "")
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Render("Taxa de Erro Máxima:"),
			fmt.Sprintf("  Valor: %s", StatusCriticalStyle.Render(fmt.Sprintf("%.2f%%", peaks.MaxErrorRate))),
			fmt.Sprintf("  HPA: %s", lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(peaks.MaxErrorRateHPA)),
			fmt.Sprintf("  Horário: %s", toLocalTime(peaks.MaxErrorRateTime).Format("15:04:05")),
		)
	}

	// Latency
	if peaks.MaxLatencyP95 > 0 {
		lines = append(lines, "")
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Render("Latência P95 Máxima:"),
			fmt.Sprintf("  Valor: %s", StatusWarningStyle.Render(fmt.Sprintf("%.0fms", peaks.MaxLatencyP95))),
			fmt.Sprintf("  HPA: %s", lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(peaks.MaxLatencyP95HPA)),
			fmt.Sprintf("  Horário: %s", toLocalTime(peaks.MaxLatencyP95Time).Format("15:04:05")),
		)
	}

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderTestIssues(result *models.StressTestMetrics) string {
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("⚠️  Problemas Detectados"))
	lines = append(lines, "")

	// Critical issues
	if len(result.CriticalIssues) > 0 {
		lines = append(lines, StatusCriticalStyle.Render(fmt.Sprintf("■ Critical (%d)", len(result.CriticalIssues))))
		for i, issue := range result.CriticalIssues {
			if i >= 5 {
				lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("  ... e mais %d problemas críticos", len(result.CriticalIssues)-5)))
				break
			}
			lines = append(lines,
				fmt.Sprintf("  %s", AnomalyTypeBadge(issue.Type)),
				fmt.Sprintf("    HPA: %s/%s/%s", issue.Cluster, issue.Namespace, issue.HPAName),
				fmt.Sprintf("    %s", lipgloss.NewStyle().Foreground(ColorTextMuted).Render(truncate(issue.Description, m.width-10))),
			)
		}
		lines = append(lines, "")
	}

	// Warning issues
	if len(result.WarningIssues) > 0 {
		lines = append(lines, StatusWarningStyle.Render(fmt.Sprintf("■ Warnings (%d)", len(result.WarningIssues))))
		for i, issue := range result.WarningIssues {
			if i >= 5 {
				lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("  ... e mais %d warnings", len(result.WarningIssues)-5)))
				break
			}
			lines = append(lines,
				fmt.Sprintf("  %s", AnomalyTypeBadge(issue.Type)),
				fmt.Sprintf("    HPA: %s/%s/%s", issue.Cluster, issue.Namespace, issue.HPAName),
				fmt.Sprintf("    %s", lipgloss.NewStyle().Foreground(ColorTextMuted).Render(truncate(issue.Description, m.width-10))),
			)
		}
	}

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderRecommendations(result *models.StressTestMetrics) string {
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("💡 Recomendações"))
	lines = append(lines, "")

	// Mostra até 5 recomendações
	max := 5
	if len(result.Recommendations) < max {
		max = len(result.Recommendations)
	}

	for i := 0; i < max; i++ {
		rec := result.Recommendations[i]

		priorityBadge := m.renderPriorityBadge(rec.Priority)
		categoryBadge := m.renderCategoryBadge(rec.Category)

		lines = append(lines,
			fmt.Sprintf("%s %s", priorityBadge, categoryBadge),
			fmt.Sprintf("  %s", lipgloss.NewStyle().Bold(true).Render(rec.Title)),
			fmt.Sprintf("  Alvo: %s", lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(rec.Target)),
			fmt.Sprintf("  %s", lipgloss.NewStyle().Foreground(ColorTextMuted).Render(rec.Description)),
		)

		if rec.Action != "" {
			lines = append(lines,
				fmt.Sprintf("  Ação: %s", MetricValueStyle.Render(rec.Action)),
			)
		}

		if i < max-1 {
			lines = append(lines, "")
		}
	}

	if len(result.Recommendations) > max {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("... e mais %d recomendações", len(result.Recommendations)-max)))
	}

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderPriorityBadge(priority models.RecommendationPriority) string {
	switch priority {
	case models.PriorityImmediate:
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#CC0000")).
			Render(" URGENTE ")
	case models.PriorityHigh:
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FF6600")).
			Render(" ALTO ")
	case models.PriorityMedium:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFCC00")).
			Render(" MÉDIO ")
	case models.PriorityLow:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Background(lipgloss.Color("#CCCCCC")).
			Render(" BAIXO ")
	default:
		return ""
	}
}

func (m Model) renderCategoryBadge(category models.RecommendationCategory) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	switch category {
	case models.CategoryScaling:
		return style.Background(lipgloss.Color("#0066CC")).Render("Scaling")
	case models.CategoryResources:
		return style.Background(lipgloss.Color("#009900")).Render("Resources")
	case models.CategoryConfiguration:
		return style.Background(lipgloss.Color("#9900CC")).Render("Config")
	case models.CategoryCode:
		return style.Background(lipgloss.Color("#CC6600")).Render("Code")
	case models.CategoryInfra:
		return style.Background(lipgloss.Color("#666666")).Render("Infra")
	default:
		return ""
	}
}

func (m Model) renderStressReportFooter() string {
	help := "Tab: Voltar ao Dashboard  •  E: Exportar Markdown  •  Shift+E: Exportar PDF  •  Q: Sair"
	return FooterStyle.Width(m.width).Render(help)
}

func StatusStyle(status models.StressTestStatus) string {
	switch status {
	case models.StressTestStatusRunning:
		return StatusOKStyle.Render("● Rodando")
	case models.StressTestStatusCompleted:
		return StatusOKStyle.Render("✓ Concluído")
	case models.StressTestStatusStopped:
		return StatusWarningStyle.Render("■ Parado")
	case models.StressTestStatusFailed:
		return StatusCriticalStyle.Render("✗ Falhou")
	default:
		return StatusInfoStyle.Render(string(status))
	}
}
