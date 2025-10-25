package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SetupStep define os passos da configuração
type SetupStep int

const (
	SetupStepMode SetupStep = iota
	SetupStepEnvironment
	SetupStepTargets
	SetupStepInterval
	SetupStepDuration
	SetupStepConfirm
	SetupStepDone
)

// SetupState estado da configuração inicial
type SetupState struct {
	currentStep SetupStep
	config      *scanner.ScanConfig

	// Para seleção de targets
	availableClusters []string
	selectedTargets   []scanner.ScanTarget
	cursorPos         int

	// Flags
	confirmed bool
}

// NewSetupState cria novo estado de setup
func NewSetupState() *SetupState {
	return &SetupState{
		currentStep:       SetupStepMode,
		config:            scanner.DefaultScanConfig(),
		availableClusters: []string{}, // TODO: Load from kubeconfig
		selectedTargets:   []scanner.ScanTarget{},
		cursorPos:         0,
		confirmed:         false,
	}
}

func (m Model) renderSetup() string {
	if m.setupState == nil {
		return "Inicializando configuração..."
	}

	var content strings.Builder

	// Header
	header := HeaderStyle.Render("🔧 HPA Watchdog - Configuração de Scan")
	content.WriteString(header + "\n\n")

	// Progress indicator
	progress := m.renderSetupProgress()
	content.WriteString(progress + "\n\n")

	// Current step
	stepContent := m.renderSetupStep()
	content.WriteString(stepContent + "\n\n")

	// Footer com help
	footer := m.renderSetupFooter()
	content.WriteString(footer)

	return content.String()
}

func (m Model) renderSetupProgress() string {
	steps := []string{
		"Modo",
		"Ambiente/Targets",
		"Intervalo",
		"Duração",
		"Confirmar",
	}

	var renderedSteps []string
	for i, step := range steps {
		style := lipgloss.NewStyle().Foreground(ColorTextMuted)

		if i < int(m.setupState.currentStep) {
			// Completo
			style = style.Foreground(ColorSuccess).Bold(true)
			step = "✓ " + step
		} else if i == int(m.setupState.currentStep) {
			// Atual
			style = style.Foreground(ColorPrimary).Bold(true)
			step = "▶ " + step
		} else {
			// Pendente
			step = "  " + step
		}

		renderedSteps = append(renderedSteps, style.Render(step))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedSteps...)
}

func (m Model) renderSetupStep() string {
	switch m.setupState.currentStep {
	case SetupStepMode:
		return m.renderStepMode()
	case SetupStepEnvironment:
		return m.renderStepEnvironment()
	case SetupStepTargets:
		return m.renderStepTargets()
	case SetupStepInterval:
		return m.renderStepInterval()
	case SetupStepDuration:
		return m.renderStepDuration()
	case SetupStepConfirm:
		return m.renderStepConfirm()
	default:
		return "Passo desconhecido"
	}
}

func (m Model) renderStepMode() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("📋 Selecione o Modo de Scan"))
	lines = append(lines, "")

	modes := []struct {
		mode        scanner.ScanMode
		title       string
		description string
	}{
		{
			mode:        scanner.ScanModeFull,
			title:       "Full - Todos os clusters de um ambiente",
			description: "Escaneia todos os clusters PRD ou HLG automaticamente",
		},
		{
			mode:        scanner.ScanModeIndividual,
			title:       "Individual - Seleção customizada",
			description: "Escolha manualmente clusters, namespaces e deployments",
		},
		{
			mode:        scanner.ScanModeStressTest,
			title:       "Stress Test - Múltiplos alvos simultâneos",
			description: "Teste de carga em múltiplos clusters/deployments",
		},
	}

	for i, mode := range modes {
		prefix := "  "
		style := TableRowStyle

		if i == m.setupState.cursorPos {
			prefix = "▶ "
			style = TableRowSelectedStyle
		}

		title := style.Render(prefix + mode.title)
		desc := lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  " + mode.description)

		lines = append(lines, title)
		lines = append(lines, desc)
		lines = append(lines, "")
	}

	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("Use ↑↓ para navegar, Enter para confirmar"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStepEnvironment() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("🌍 Selecione o Ambiente"))
	lines = append(lines, "")

	environments := []struct {
		env         scanner.Environment
		title       string
		description string
	}{
		{
			env:         scanner.EnvironmentPRD,
			title:       "PRD - Produção",
			description: "Todos os clusters *-prd-admin",
		},
		{
			env:         scanner.EnvironmentHLG,
			title:       "HLG - Homologação",
			description: "Todos os clusters *-hlg-admin",
		},
	}

	for i, env := range environments {
		prefix := "  "
		style := TableRowStyle

		if i == m.setupState.cursorPos {
			prefix = "▶ "
			style = TableRowSelectedStyle
		}

		title := style.Render(prefix + env.title)
		desc := lipgloss.NewStyle().Foreground(ColorTextMuted).Render("  " + env.description)

		lines = append(lines, title)
		lines = append(lines, desc)
		lines = append(lines, "")
	}

	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("Use ↑↓ para navegar, Enter para confirmar"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStepTargets() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("🎯 Selecione os Targets"))
	lines = append(lines, "")

	// Lista de clusters disponíveis
	lines = append(lines, TableHeaderStyle.Render("Clusters Disponíveis:"))
	lines = append(lines, "")

	if len(m.setupState.availableClusters) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Carregando clusters do kubeconfig..."))
	} else {
		for i, cluster := range m.setupState.availableClusters {
			prefix := "[ ]"
			style := TableRowStyle

			// Verifica se está selecionado
			selected := false
			for _, target := range m.setupState.selectedTargets {
				if target.Cluster == cluster {
					selected = true
					break
				}
			}

			if selected {
				prefix = "[✓]"
				style = style.Foreground(ColorSuccess)
			}

			if i == m.setupState.cursorPos {
				prefix = "▶ " + prefix
				style = TableRowSelectedStyle
			} else {
				prefix = "  " + prefix
			}

			lines = append(lines, style.Render(prefix+" "+cluster))
		}
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Selecionados: %d", len(m.setupState.selectedTargets)))
	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("↑↓: Navegar  Space: Selecionar  Enter: Confirmar  A: Selecionar todos"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStepInterval() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("⏱️  Intervalo entre Scans"))
	lines = append(lines, "")

	intervals := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		60 * time.Minute,
	}

	for i, interval := range intervals {
		prefix := "  "
		style := TableRowStyle

		if i == m.setupState.cursorPos {
			prefix = "▶ "
			style = TableRowSelectedStyle
		}

		label := fmt.Sprintf("%v", interval)
		lines = append(lines, style.Render(prefix+label))
	}

	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("↑↓: Navegar  Enter: Confirmar"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStepDuration() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("⏳ Duração do Teste"))
	lines = append(lines, "")

	durations := []struct {
		duration    time.Duration
		description string
	}{
		{0, "Infinito (até Ctrl+C)"},
		{15 * time.Minute, "15 minutos"},
		{30 * time.Minute, "30 minutos"},
		{1 * time.Hour, "1 hora"},
		{2 * time.Hour, "2 horas"},
		{3 * time.Hour, "3 horas (máximo)"},
	}

	for i, dur := range durations {
		prefix := "  "
		style := TableRowStyle

		if i == m.setupState.cursorPos {
			prefix = "▶ "
			style = TableRowSelectedStyle
		}

		scans := ""
		if dur.duration > 0 && m.setupState.config.Interval > 0 {
			estimated := int(dur.duration / m.setupState.config.Interval)
			scans = fmt.Sprintf(" (~%d scans)", estimated)
		}

		lines = append(lines, style.Render(prefix+dur.description+scans))
	}

	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("↑↓: Navegar  Enter: Confirmar"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderStepConfirm() string {
	var lines []string

	lines = append(lines, SectionTitleStyle.Render("✅ Confirmar Configuração"))
	lines = append(lines, "")

	// Resumo da configuração
	summary := m.setupState.config.Summary()
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorTextPrimary).Render(summary))

	lines = append(lines, "")
	lines = append(lines, StatusWarningStyle.Render("⚠️  O scan será iniciado imediatamente após confirmação"))
	lines = append(lines, "")

	// Opções
	options := []string{
		"▶ Iniciar Scan",
		"  Voltar e Ajustar",
		"  Cancelar",
	}

	for i, opt := range options {
		style := TableRowStyle
		if i == 0 {
			style = TableRowSelectedStyle.Copy().Foreground(ColorSuccess)
		}
		lines = append(lines, style.Render(opt))
	}

	lines = append(lines, "")
	lines = append(lines, HelpStyle.Render("Enter: Iniciar  Backspace: Voltar  Esc: Cancelar"))

	return BoxStyle.Copy().Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderSetupFooter() string {
	help := "ESC: Cancelar configuração  •  Backspace: Voltar passo"
	return FooterStyle.Copy().Width(m.width).Render(help)
}

// handleSetupKeyPress processa teclas na view de setup
func (m Model) handleSetupKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.setupState == nil {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Cancelar setup
		return m, tea.Quit

	case "backspace":
		// Voltar passo
		if m.setupState.currentStep > SetupStepMode {
			m.setupState.currentStep--
			m.setupState.cursorPos = 0
		}
		return m, nil

	case "up", "k":
		if m.setupState.cursorPos > 0 {
			m.setupState.cursorPos--
		}
		return m, nil

	case "down", "j":
		maxPos := m.getMaxSetupCursorPos()
		if m.setupState.cursorPos < maxPos {
			m.setupState.cursorPos++
		}
		return m, nil

	case "enter":
		return m.handleSetupSelect()

	case " ": // Space para multi-select (targets)
		if m.setupState.currentStep == SetupStepTargets {
			return m.toggleTargetSelection()
		}
		return m, nil

	case "a", "A": // Selecionar todos (targets)
		if m.setupState.currentStep == SetupStepTargets {
			return m.selectAllTargets()
		}
		return m, nil
	}

	return m, nil
}

// handleSetupSelect processa Enter no setup
func (m Model) handleSetupSelect() (tea.Model, tea.Cmd) {
	if m.setupState == nil {
		return m, nil
	}

	switch m.setupState.currentStep {
	case SetupStepMode:
		// Seleciona modo baseado no cursor
		m.setupState.config.Mode = scanner.ScanMode(m.setupState.cursorPos)
		m.setupState.currentStep = SetupStepEnvironment
		m.setupState.cursorPos = 0

	case SetupStepEnvironment:
		// Seleciona ambiente (apenas para modo Full)
		if m.setupState.config.Mode == scanner.ScanModeFull {
			if m.setupState.cursorPos == 0 {
				m.setupState.config.Environment = scanner.EnvironmentPRD
			} else {
				m.setupState.config.Environment = scanner.EnvironmentHLG
			}
		}
		m.setupState.currentStep = SetupStepTargets
		m.setupState.cursorPos = 0

	case SetupStepTargets:
		// Confirma targets selecionados
		if len(m.setupState.selectedTargets) > 0 || m.setupState.config.Mode == scanner.ScanModeFull {
			m.setupState.currentStep = SetupStepInterval
			m.setupState.cursorPos = 0
		}

	case SetupStepInterval:
		// Seleciona intervalo
		intervals := []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			10 * time.Minute,
			15 * time.Minute,
			30 * time.Minute,
			60 * time.Minute,
		}
		if m.setupState.cursorPos < len(intervals) {
			m.setupState.config.Interval = intervals[m.setupState.cursorPos]
		}
		m.setupState.currentStep = SetupStepDuration
		m.setupState.cursorPos = 0

	case SetupStepDuration:
		// Seleciona duração
		durations := []time.Duration{
			0, // Infinito
			15 * time.Minute,
			30 * time.Minute,
			1 * time.Hour,
			2 * time.Hour,
			3 * time.Hour,
		}
		if m.setupState.cursorPos < len(durations) {
			m.setupState.config.Duration = durations[m.setupState.cursorPos]
			m.setupState.config.CalculateEndTime()
		}
		m.setupState.currentStep = SetupStepConfirm
		m.setupState.cursorPos = 0

	case SetupStepConfirm:
		// Confirma e inicia scan
		m.setupState.confirmed = true
		m.setupState.currentStep = SetupStepDone
		m.currentView = ViewDashboard // Muda para dashboard
		// TODO: Iniciar scan engine aqui
	}

	return m, nil
}

// getMaxSetupCursorPos retorna posição máxima do cursor para o step atual
func (m Model) getMaxSetupCursorPos() int {
	if m.setupState == nil {
		return 0
	}

	switch m.setupState.currentStep {
	case SetupStepMode:
		return 2 // 3 modos (0, 1, 2)

	case SetupStepEnvironment:
		return 1 // 2 ambientes (0, 1)

	case SetupStepTargets:
		if len(m.setupState.availableClusters) > 0 {
			return len(m.setupState.availableClusters) - 1
		}
		return 0

	case SetupStepInterval:
		return 5 // 6 opções (0-5)

	case SetupStepDuration:
		return 5 // 6 opções (0-5)

	case SetupStepConfirm:
		return 2 // 3 opções (Iniciar, Voltar, Cancelar)

	default:
		return 0
	}
}

// toggleTargetSelection alterna seleção de um target
func (m Model) toggleTargetSelection() (tea.Model, tea.Cmd) {
	if m.setupState == nil || len(m.setupState.availableClusters) == 0 {
		return m, nil
	}

	if m.setupState.cursorPos >= len(m.setupState.availableClusters) {
		return m, nil
	}

	cluster := m.setupState.availableClusters[m.setupState.cursorPos]

	// Verifica se já está selecionado
	found := -1
	for i, target := range m.setupState.selectedTargets {
		if target.Cluster == cluster {
			found = i
			break
		}
	}

	if found >= 0 {
		// Remove da seleção
		m.setupState.selectedTargets = append(
			m.setupState.selectedTargets[:found],
			m.setupState.selectedTargets[found+1:]...,
		)
	} else {
		// Adiciona à seleção
		m.setupState.selectedTargets = append(m.setupState.selectedTargets, scanner.ScanTarget{
			Cluster:     cluster,
			Namespaces:  []string{}, // Todos
			Deployments: []string{}, // Todos
			HPAs:        []string{}, // Todos
		})
	}

	return m, nil
}

// selectAllTargets seleciona todos os targets
func (m Model) selectAllTargets() (tea.Model, tea.Cmd) {
	if m.setupState == nil {
		return m, nil
	}

	// Limpa seleção atual
	m.setupState.selectedTargets = []scanner.ScanTarget{}

	// Adiciona todos os clusters
	for _, cluster := range m.setupState.availableClusters {
		m.setupState.selectedTargets = append(m.setupState.selectedTargets, scanner.ScanTarget{
			Cluster:     cluster,
			Namespaces:  []string{}, // Todos
			Deployments: []string{}, // Todos
			HPAs:        []string{}, // Todos
		})
	}

	return m, nil
}
