package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
)

// Período de análise histórica
type HistoryPeriod int

const (
	Period1Hour HistoryPeriod = iota
	Period3Hours
	Period6Hours
	Period24Hours
	Period7Days
)

func (p HistoryPeriod) String() string {
	switch p {
	case Period1Hour:
		return "Última 1h"
	case Period3Hours:
		return "Últimas 3h"
	case Period6Hours:
		return "Últimas 6h"
	case Period24Hours:
		return "Últimas 24h"
	case Period7Days:
		return "Últimos 7d"
	default:
		return "Desconhecido"
	}
}

func (p HistoryPeriod) Duration() time.Duration {
	switch p {
	case Period1Hour:
		return 1 * time.Hour
	case Period3Hours:
		return 3 * time.Hour
	case Period6Hours:
		return 6 * time.Hour
	case Period24Hours:
		return 24 * time.Hour
	case Period7Days:
		return 7 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// Estado da view de histórico (adicionar ao Model depois)
type HistoryState struct {
	selectedCluster   string
	selectedNamespace string
	selectedHPA       string
	period            HistoryPeriod
}

func (m Model) renderHistory() string {
	var content strings.Builder

	// Header
	header := m.renderHeader("📊 HPA Watchdog - Análise Histórica")
	content.WriteString(header + "\n\n")

	// Tabs
	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	// Se não houver dados, mostra mensagem
	if len(m.snapshots) == 0 {
		emptyMsg := BoxStyle.Width(m.width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"",
				StatusInfoStyle.Render("📊 Nenhum dado histórico disponível"),
				"",
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Os dados aparecerão após a coleta de snapshots."),
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render("A persistência SQLite está habilitada e salvará dados automaticamente."),
				"",
			),
		)
		content.WriteString(emptyMsg + "\n\n")

		// Footer
		footer := m.renderFooter()
		content.WriteString(footer)
		return content.String()
	}

	// Seleciona primeiro HPA disponível para análise (pode ser melhorado com navegação)
	var selectedTS *models.TimeSeriesData
	for _, ts := range m.snapshots {
		selectedTS = ts
		break
	}

	if selectedTS == nil {
		content.WriteString("Erro: TimeSeriesData não disponível\n")
		return content.String()
	}

	// Extrai informações do snapshot mais recente
	latest := selectedTS.GetLatest()
	if latest == nil {
		content.WriteString("Erro: Snapshot mais recente não disponível\n")
		return content.String()
	}

	// Info box: Cluster/HPA selecionado
	infoBox := m.renderHistoryInfoBox(latest, selectedTS)
	content.WriteString(infoBox + "\n\n")

	// Gráfico de CPU
	cpuGraph := m.renderCPUGraph(selectedTS)
	content.WriteString(cpuGraph + "\n\n")

	// Gráfico de Réplicas
	replicaGraph := m.renderReplicaGraph(selectedTS)
	content.WriteString(replicaGraph + "\n\n")

	// Tabela de comparação com baseline
	baselineTable := m.renderBaselineComparison(selectedTS)
	content.WriteString(baselineTable + "\n\n")

	// Footer com ajuda
	footer := m.renderHistoryFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderHistoryInfoBox(latest *models.HPASnapshot, ts *models.TimeSeriesData) string {
	cluster := latest.Cluster
	namespace := latest.Namespace
	hpa := latest.Name
	period := "Últimas 24h" // Por enquanto fixo
	dataPoints := len(ts.Snapshots)

	leftInfo := lipgloss.JoinVertical(lipgloss.Left,
		MetricLabelStyle.Render("Cluster: ")+MetricValueStyle.Render(cluster),
		MetricLabelStyle.Render("HPA: ")+MetricValueStyle.Render(fmt.Sprintf("%s/%s", namespace, hpa)),
	)

	rightInfo := lipgloss.JoinVertical(lipgloss.Left,
		MetricLabelStyle.Render("Período: ")+MetricValueStyle.Render(period),
		MetricLabelStyle.Render("Dados: ")+MetricValueStyle.Render(fmt.Sprintf("%d snapshots", dataPoints)),
	)

	infoLine := lipgloss.JoinHorizontal(lipgloss.Top,
		leftInfo,
		strings.Repeat(" ", 10),
		rightInfo,
	)

	return BoxStyle.Width(m.width - 4).Render(infoLine)
}

func (m Model) renderCPUGraph(ts *models.TimeSeriesData) string {
	snapshots := ts.Snapshots
	if len(snapshots) == 0 {
		return BoxStyle.Width(m.width - 4).Render("Sem dados de CPU")
	}

	// Extrai dados de CPU
	cpuData := make([]float64, 0, len(snapshots))
	for _, s := range snapshots {
		cpuData = append(cpuData, s.CPUCurrent)
	}

	// Gera gráfico ASCII
	graphWidth := m.width - 10
	if graphWidth < 40 {
		graphWidth = 40
	}

	graph := asciigraph.Plot(cpuData,
		asciigraph.Height(10),
		asciigraph.Width(graphWidth),
		asciigraph.Caption("CPU Usage (%)"),
	)

	// Adiciona estatísticas
	stats := ts.Stats
	statsLine := fmt.Sprintf("Min: %.1f%%  │  Max: %.1f%%  │  Atual: %.1f%%  │  Média: %.1f%%  │  Trend: %s",
		stats.CPUMin,
		stats.CPUMax,
		cpuData[len(cpuData)-1],
		stats.CPUAverage,
		stats.CPUTrend,
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		SectionTitleStyle.Render("📈 CPU Usage (%)"),
		"",
		graph,
		"",
		lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(statsLine),
	)

	return BoxStyle.Width(m.width - 4).Render(content)
}

func (m Model) renderReplicaGraph(ts *models.TimeSeriesData) string {
	snapshots := ts.Snapshots
	if len(snapshots) == 0 {
		return BoxStyle.Width(m.width - 4).Render("Sem dados de réplicas")
	}

	// Extrai dados de réplicas
	replicaData := make([]float64, 0, len(snapshots))
	for _, s := range snapshots {
		replicaData = append(replicaData, float64(s.CurrentReplicas))
	}

	// Gera gráfico ASCII
	graphWidth := m.width - 10
	if graphWidth < 40 {
		graphWidth = 40
	}

	graph := asciigraph.Plot(replicaData,
		asciigraph.Height(8),
		asciigraph.Width(graphWidth),
		asciigraph.Caption("Réplicas"),
	)

	// Calcula estatísticas de réplicas
	var minRep, maxRep int32 = 999999, 0
	var sumRep int64 = 0
	for _, s := range snapshots {
		if s.CurrentReplicas < minRep {
			minRep = s.CurrentReplicas
		}
		if s.CurrentReplicas > maxRep {
			maxRep = s.CurrentReplicas
		}
		sumRep += int64(s.CurrentReplicas)
	}
	avgRep := float64(sumRep) / float64(len(snapshots))

	latest := snapshots[len(snapshots)-1]
	statsLine := fmt.Sprintf("Min: %d  │  Max: %d  │  Atual: %d  │  Média: %.1f  │  Mudanças: %d",
		minRep,
		maxRep,
		latest.CurrentReplicas,
		avgRep,
		ts.Stats.ReplicaChanges,
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		SectionTitleStyle.Render("🔢 Réplicas"),
		"",
		graph,
		"",
		lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(statsLine),
	)

	return BoxStyle.Width(m.width - 4).Render(content)
}

func (m Model) renderBaselineComparison(ts *models.TimeSeriesData) string {
	if len(ts.Snapshots) == 0 {
		return BoxStyle.Width(m.width - 4).Render("Sem dados para comparação")
	}

	latest := ts.GetLatest()
	stats := ts.Stats

	// Tabela de comparação
	var lines []string
	lines = append(lines, SectionTitleStyle.Render("📊 Comparação com Baseline"))
	lines = append(lines, "")

	// Header da tabela
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableHeaderStyle.Copy().Width(20).Render("Métrica"),
		TableHeaderStyle.Copy().Width(15).Render("Atual"),
		TableHeaderStyle.Copy().Width(15).Render("Baseline"),
		TableHeaderStyle.Copy().Width(20).Render("Desvio"),
		TableHeaderStyle.Copy().Width(10).Render("Status"),
	)
	lines = append(lines, headerLine)
	lines = append(lines, Divider(m.width-6))

	// CPU
	cpuDeviation := latest.CPUCurrent - stats.CPUAverage
	cpuStatus := m.renderDeviationStatus(cpuDeviation, 10.0) // threshold: 10%
	cpuLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableRowStyle.Copy().Width(20).Render("CPU"),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%.1f%%", latest.CPUCurrent)),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%.1f%%", stats.CPUAverage)),
		TableRowStyle.Copy().Width(20).Render(fmt.Sprintf("%+.1f%%", cpuDeviation)),
		TableRowStyle.Copy().Width(10).Render(cpuStatus),
	)
	lines = append(lines, cpuLine)

	// Memory
	memDeviation := latest.MemoryCurrent - stats.MemoryAverage
	memStatus := m.renderDeviationStatus(memDeviation, 10.0)
	memLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableRowStyle.Copy().Width(20).Render("Memory"),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%.1f%%", latest.MemoryCurrent)),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%.1f%%", stats.MemoryAverage)),
		TableRowStyle.Copy().Width(20).Render(fmt.Sprintf("%+.1f%%", memDeviation)),
		TableRowStyle.Copy().Width(10).Render(memStatus),
	)
	lines = append(lines, memLine)

	// Réplicas - Calcula média manualmente
	var sumRep int64 = 0
	for _, s := range ts.Snapshots {
		sumRep += int64(s.CurrentReplicas)
	}
	repBaseline := float64(sumRep) / float64(len(ts.Snapshots))
	repDeviation := float64(latest.CurrentReplicas) - repBaseline
	repStatus := m.renderDeviationStatus(repDeviation, 2.0) // threshold: 2 réplicas
	repLine := lipgloss.JoinHorizontal(lipgloss.Top,
		TableRowStyle.Copy().Width(20).Render("Réplicas"),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%d", latest.CurrentReplicas)),
		TableRowStyle.Copy().Width(15).Render(fmt.Sprintf("%.1f", repBaseline)),
		TableRowStyle.Copy().Width(20).Render(fmt.Sprintf("%+.1f", repDeviation)),
		TableRowStyle.Copy().Width(10).Render(repStatus),
	)
	lines = append(lines, repLine)

	return BoxStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderDeviationStatus(deviation, threshold float64) string {
	absDeviation := deviation
	if absDeviation < 0 {
		absDeviation = -absDeviation
	}

	if absDeviation > threshold*2 {
		return StatusCriticalStyle.Render("🔴")
	} else if absDeviation > threshold {
		return StatusWarningStyle.Render("⚠️")
	} else {
		return StatusOKStyle.Render("✓")
	}
}

func (m Model) renderHistoryFooter() string {
	help := "Tab: Mudar view  •  ↑↓/jk: Navegar HPAs  •  1-9: Período  •  H/Home: Dashboard  •  Q: Sair"

	// Adiciona status de scan e tecla P se scan estiver rodando
	if m.scanRunning {
		if m.scanPaused {
			help += "  •  P: Retomar scan"
		} else {
			help += "  •  P: Pausar scan"
		}
	}

	return FooterStyle.Width(m.width).Render(help)
}
