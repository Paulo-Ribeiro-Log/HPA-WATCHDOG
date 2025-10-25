package tui

import (
	"testing"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSetupState(t *testing.T) {
	state := NewSetupState()

	if state == nil {
		t.Fatal("NewSetupState retornou nil")
	}

	if state.currentStep != SetupStepMode {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepMode, state.currentStep)
	}

	if state.config == nil {
		t.Fatal("config não deve ser nil")
	}

	if state.confirmed {
		t.Error("confirmed deveria ser false inicialmente")
	}
}

func TestSetupKeyNavigation(t *testing.T) {
	m := New()

	// Deve iniciar no ViewSetup
	if m.currentView != ViewSetup {
		t.Errorf("currentView esperado: %v, obtido: %v", ViewSetup, m.currentView)
	}

	// Testa navegação para baixo
	newModel, _ := m.handleSetupKeyPress(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.setupState.cursorPos != 1 {
		t.Errorf("cursorPos esperado: 1, obtido: %d", m.setupState.cursorPos)
	}

	// Testa navegação para cima
	newModel, _ = m.handleSetupKeyPress(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	if m.setupState.cursorPos != 0 {
		t.Errorf("cursorPos esperado: 0, obtido: %d", m.setupState.cursorPos)
	}
}

func TestSetupModeSelection(t *testing.T) {
	m := New()

	// Seleciona modo Full (cursor em 0)
	m.setupState.cursorPos = 0
	newModel, _ := m.handleSetupSelect()
	m = newModel.(Model)

	if m.setupState.config.Mode != scanner.ScanModeFull {
		t.Errorf("Mode esperado: %v, obtido: %v", scanner.ScanModeFull, m.setupState.config.Mode)
	}

	if m.setupState.currentStep != SetupStepEnvironment {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepEnvironment, m.setupState.currentStep)
	}
}

func TestSetupEnvironmentSelection(t *testing.T) {
	m := New()

	// Avança para SetupStepEnvironment
	m.setupState.currentStep = SetupStepEnvironment
	m.setupState.config.Mode = scanner.ScanModeFull

	// Seleciona PRD (cursor em 0)
	m.setupState.cursorPos = 0
	newModel, _ := m.handleSetupSelect()
	m = newModel.(Model)

	if m.setupState.config.Environment != scanner.EnvironmentPRD {
		t.Errorf("Environment esperado: %v, obtido: %v", scanner.EnvironmentPRD, m.setupState.config.Environment)
	}

	if m.setupState.currentStep != SetupStepTargets {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepTargets, m.setupState.currentStep)
	}
}

func TestSetupBackspace(t *testing.T) {
	m := New()

	// Avança para outro step
	m.setupState.currentStep = SetupStepEnvironment

	// Aperta backspace
	newModel, _ := m.handleSetupKeyPress(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(Model)

	if m.setupState.currentStep != SetupStepMode {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepMode, m.setupState.currentStep)
	}
}

func TestToggleTargetSelection(t *testing.T) {
	m := New()
	m.setupState.availableClusters = []string{
		"cluster-1",
		"cluster-2",
		"cluster-3",
	}
	m.setupState.cursorPos = 0

	// Seleciona cluster-1
	newModel, _ := m.toggleTargetSelection()
	m = newModel.(Model)

	if len(m.setupState.selectedTargets) != 1 {
		t.Errorf("selectedTargets esperado: 1, obtido: %d", len(m.setupState.selectedTargets))
	}

	if m.setupState.selectedTargets[0].Cluster != "cluster-1" {
		t.Errorf("Cluster selecionado esperado: cluster-1, obtido: %s", m.setupState.selectedTargets[0].Cluster)
	}

	// Deseleciona cluster-1 (toggle novamente)
	newModel, _ = m.toggleTargetSelection()
	m = newModel.(Model)

	if len(m.setupState.selectedTargets) != 0 {
		t.Errorf("selectedTargets esperado: 0, obtido: %d", len(m.setupState.selectedTargets))
	}
}

func TestSelectAllTargets(t *testing.T) {
	m := New()
	m.setupState.availableClusters = []string{
		"cluster-1",
		"cluster-2",
		"cluster-3",
	}

	// Seleciona todos
	newModel, _ := m.selectAllTargets()
	m = newModel.(Model)

	if len(m.setupState.selectedTargets) != 3 {
		t.Errorf("selectedTargets esperado: 3, obtido: %d", len(m.setupState.selectedTargets))
	}

	// Verifica que todos os clusters foram selecionados
	for i, cluster := range m.setupState.availableClusters {
		if m.setupState.selectedTargets[i].Cluster != cluster {
			t.Errorf("Cluster %d esperado: %s, obtido: %s", i, cluster, m.setupState.selectedTargets[i].Cluster)
		}
	}
}

func TestGetMaxSetupCursorPos(t *testing.T) {
	m := New()

	tests := []struct {
		step     SetupStep
		expected int
	}{
		{SetupStepMode, 2},        // 3 modos
		{SetupStepEnvironment, 1}, // 2 ambientes
		{SetupStepInterval, 5},    // 6 intervalos
		{SetupStepDuration, 5},    // 6 durações
		{SetupStepConfirm, 2},     // 3 opções
	}

	for _, tt := range tests {
		m.setupState.currentStep = tt.step
		result := m.getMaxSetupCursorPos()

		if result != tt.expected {
			t.Errorf("Step %v: maxCursorPos esperado: %d, obtido: %d", tt.step, tt.expected, result)
		}
	}
}

func TestSetupIntervalSelection(t *testing.T) {
	m := New()
	m.setupState.currentStep = SetupStepInterval
	m.setupState.cursorPos = 1 // Seleciona 5 minutos

	newModel, _ := m.handleSetupSelect()
	m = newModel.(Model)

	expected := int64(5 * 60 * 1_000_000_000) // 5 minutos em nanosegundos
	if int64(m.setupState.config.Interval) != expected {
		t.Errorf("Interval esperado: %d, obtido: %d", expected, m.setupState.config.Interval)
	}

	if m.setupState.currentStep != SetupStepDuration {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepDuration, m.setupState.currentStep)
	}
}

func TestSetupDurationSelection(t *testing.T) {
	m := New()
	m.setupState.currentStep = SetupStepDuration
	m.setupState.cursorPos = 3 // Seleciona 1 hora

	newModel, _ := m.handleSetupSelect()
	m = newModel.(Model)

	expected := int64(1 * 60 * 60 * 1_000_000_000) // 1 hora em nanosegundos
	if int64(m.setupState.config.Duration) != expected {
		t.Errorf("Duration esperado: %d, obtido: %d", expected, m.setupState.config.Duration)
	}

	if m.setupState.currentStep != SetupStepConfirm {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepConfirm, m.setupState.currentStep)
	}
}

func TestSetupConfirmation(t *testing.T) {
	m := New()
	m.setupState.currentStep = SetupStepConfirm

	newModel, _ := m.handleSetupSelect()
	m = newModel.(Model)

	if !m.setupState.confirmed {
		t.Error("confirmed deveria ser true após confirmação")
	}

	if m.setupState.currentStep != SetupStepDone {
		t.Errorf("currentStep esperado: %v, obtido: %v", SetupStepDone, m.setupState.currentStep)
	}

	if m.currentView != ViewDashboard {
		t.Errorf("currentView esperado: %v, obtido: %v", ViewDashboard, m.currentView)
	}
}

func TestRenderSetup(t *testing.T) {
	m := New()
	m.ready = true
	m.width = 120
	m.height = 40

	// Deve renderizar sem panic
	output := m.renderSetup()

	if output == "" {
		t.Error("renderSetup não deveria retornar string vazia")
	}

	// Deve conter título (verificação básica)
	if len(output) < 10 {
		t.Error("renderSetup deveria retornar conteúdo com pelo menos 10 caracteres")
	}
}
