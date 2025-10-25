package tui

import "github.com/charmbracelet/lipgloss"

// Cores do tema
var (
	// Cores principais
	ColorPrimary   = lipgloss.Color("#00D9FF") // Cyan brilhante
	ColorSecondary = lipgloss.Color("#7C3AED") // Purple
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Orange
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorInfo      = lipgloss.Color("#3B82F6") // Blue
	ColorMuted     = lipgloss.Color("#6B7280") // Gray

	// Cores de fundo
	ColorBgPrimary   = lipgloss.Color("#1E293B") // Slate dark
	ColorBgSecondary = lipgloss.Color("#334155") // Slate
	ColorBgHighlight = lipgloss.Color("#475569") // Slate light

	// Cores de texto
	ColorTextPrimary   = lipgloss.Color("#F8FAFC") // White
	ColorTextSecondary = lipgloss.Color("#CBD5E1") // Gray light
	ColorTextMuted     = lipgloss.Color("#94A3B8") // Gray
)

// Estilos base
var (
	// Header principal
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	// Título de seção
	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				Padding(0, 1)

	// Box containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBgSecondary).
			Padding(1, 2)

	BoxHighlightStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(1, 2)

	// Tabs
	TabStyle = lipgloss.NewStyle().
			Foreground(ColorTextSecondary).
			Padding(0, 2)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 2).
			Underline(true)

	// Status badges
	StatusOKStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	StatusCriticalStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	// Table headers
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 1)

	// Table rows
	TableRowStyle = lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Padding(0, 1)

	TableRowSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Background(ColorBgSecondary).
				Bold(true).
				Padding(0, 1)

	// Metrics
	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(ColorTextSecondary).
				Width(20)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Bold(true)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true).
			MarginTop(1)

	// Footer
	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorTextSecondary).
			Background(ColorBgSecondary).
			Padding(0, 1)
)

// Helper functions para severity badges
func SeverityBadge(severity string) string {
	switch severity {
	case "Critical":
		return StatusCriticalStyle.Render("🔴 CRITICAL")
	case "Warning":
		return StatusWarningStyle.Render("🟡 WARNING")
	case "Info":
		return StatusInfoStyle.Render("🔵 INFO")
	default:
		return StatusInfoStyle.Render("• " + severity)
	}
}

// Helper para status de clusters
func ClusterStatusBadge(status string) string {
	switch status {
	case "Online":
		return StatusOKStyle.Render("🟢 Online")
	case "Offline":
		return StatusCriticalStyle.Render("🔴 Offline")
	case "Error":
		return StatusWarningStyle.Render("🟡 Error")
	default:
		return StatusInfoStyle.Render("• " + status)
	}
}

// Helper para anomaly types
func AnomalyTypeBadge(anomalyType string) string {
	colors := map[string]lipgloss.Color{
		"OSCILLATION":     ColorWarning,
		"MAXED_OUT":       ColorDanger,
		"OOM_KILLED":      ColorDanger,
		"PODS_NOT_READY":  ColorWarning,
		"HIGH_ERROR_RATE": ColorDanger,
		"CPU_SPIKE":       ColorWarning,
		"REPLICA_SPIKE":   ColorWarning,
		"ERROR_SPIKE":     ColorDanger,
		"LATENCY_SPIKE":   ColorWarning,
		"CPU_DROP":        ColorInfo,
	}

	color, ok := colors[anomalyType]
	if !ok {
		color = ColorMuted
	}

	style := lipgloss.NewStyle().Foreground(color).Bold(true)
	return style.Render(anomalyType)
}

// Helper para renderizar métricas
func RenderMetric(label, value string) string {
	return MetricLabelStyle.Render(label) + " " + MetricValueStyle.Render(value)
}

// Helper para criar divisores
func Divider(width int) string {
	return lipgloss.NewStyle().
		Foreground(ColorBgSecondary).
		Render(lipgloss.NewStyle().Width(width).Render("─"))
}
