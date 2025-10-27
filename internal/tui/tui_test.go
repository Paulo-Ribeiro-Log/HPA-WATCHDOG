package tui

import (
	"testing"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/analyzer"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
)

func TestNew(t *testing.T) {
	model := New()

	if model.currentView != ViewSetup {
		t.Errorf("Expected initial view to be Setup, got %v", model.currentView)
	}

	if model.setupState == nil {
		t.Error("Expected setupState to be initialized, got nil")
	}

	if model.filterSeverity != "All" {
		t.Errorf("Expected initial filter to be 'All', got %s", model.filterSeverity)
	}

	if !model.autoRefresh {
		t.Error("Expected autoRefresh to be enabled")
	}

	if model.snapshots == nil {
		t.Error("Expected snapshots map to be initialized")
	}

	if model.anomalies == nil {
		t.Error("Expected anomalies slice to be initialized")
	}

	if model.clusters == nil {
		t.Error("Expected clusters map to be initialized")
	}
}

func TestHandleSnapshot(t *testing.T) {
	model := New()

	snapshot := &models.HPASnapshot{
		Timestamp:       time.Now(),
		Cluster:         "test-cluster",
		Namespace:       "default",
		Name:            "test-hpa",
		CurrentReplicas: 3,
		MinReplicas:     2,
		MaxReplicas:     10,
		CPUCurrent:      50.0,
	}

	model.handleSnapshot(snapshot)

	// Verifica que snapshot foi armazenado
	key := makeKey(snapshot.Cluster, snapshot.Namespace, snapshot.Name)
	if _, exists := model.snapshots[key]; !exists {
		t.Error("Expected snapshot to be stored")
	}

	// Verifica que cluster foi criado
	if cluster, exists := model.clusters[snapshot.Cluster]; !exists {
		t.Error("Expected cluster to be created")
	} else {
		if cluster.Name != snapshot.Cluster {
			t.Errorf("Expected cluster name %s, got %s", snapshot.Cluster, cluster.Name)
		}
		if cluster.Status != "Online" {
			t.Errorf("Expected cluster status Online, got %s", cluster.Status)
		}
	}
}

func TestHandleAnomaly(t *testing.T) {
	model := New()

	anomaly := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeCPUSpike,
		Severity:  models.SeverityWarning,
		Cluster:   "test-cluster",
		Namespace: "default",
		HPAName:   "test-hpa",
		Timestamp: time.Now(),
		Message:   "CPU spike detected",
	}

	model.handleAnomaly(anomaly)

	// Verifica que anomalia foi adicionada
	if len(model.anomalies) != 1 {
		t.Errorf("Expected 1 anomaly, got %d", len(model.anomalies))
	}

	// Verifica que anomalia está no início da lista (mais recente)
	if model.anomalies[0].Type != analyzer.AnomalyTypeCPUSpike {
		t.Errorf("Expected CPU_SPIKE anomaly, got %s", model.anomalies[0].Type)
	}
}

func TestGetFilteredAnomalies(t *testing.T) {
	model := New()

	// Adiciona anomalias de diferentes severidades
	anomalies := []analyzer.Anomaly{
		{
			Type:     analyzer.AnomalyTypeCPUSpike,
			Severity: models.SeverityCritical,
			Cluster:  "cluster1",
		},
		{
			Type:     analyzer.AnomalyTypeOscillation,
			Severity: models.SeverityWarning,
			Cluster:  "cluster1",
		},
		{
			Type:     analyzer.AnomalyTypeMaxedOut,
			Severity: models.SeverityCritical,
			Cluster:  "cluster2",
		},
	}

	for _, a := range anomalies {
		model.handleAnomaly(a)
	}

	// Testa filtro "All"
	model.filterSeverity = "All"
	model.filterCluster = ""
	filtered := model.getFilteredAnomalies()
	if len(filtered) != 3 {
		t.Errorf("Expected 3 anomalies with 'All' filter, got %d", len(filtered))
	}

	// Testa filtro "Critical"
	model.filterSeverity = "Critical"
	filtered = model.getFilteredAnomalies()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 critical anomalies, got %d", len(filtered))
	}

	// Testa filtro "Warning"
	model.filterSeverity = "Warning"
	filtered = model.getFilteredAnomalies()
	if len(filtered) != 1 {
		t.Errorf("Expected 1 warning anomaly, got %d", len(filtered))
	}

	// Testa filtro de cluster
	model.filterSeverity = "All"
	model.filterCluster = "cluster1"
	filtered = model.getFilteredAnomalies()
	if len(filtered) != 2 {
		t.Errorf("Expected 2 anomalies from cluster1, got %d", len(filtered))
	}
}

func TestMakeKey(t *testing.T) {
	key := makeKey("cluster", "namespace", "hpa")
	expected := "cluster/namespace/hpa"

	if key != expected {
		t.Errorf("Expected key %s, got %s", expected, key)
	}
}

func TestRenderDashboard(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.ready = true

	// Adiciona dados de teste
	snapshot := &models.HPASnapshot{
		Timestamp: time.Now(),
		Cluster:   "test-cluster",
		Namespace: "default",
		Name:      "test-hpa",
	}
	model.handleSnapshot(snapshot)

	anomaly := analyzer.Anomaly{
		Type:     analyzer.AnomalyTypeCPUSpike,
		Severity: models.SeverityWarning,
		Cluster:  "test-cluster",
		Message:  "Test anomaly",
	}
	model.handleAnomaly(anomaly)

	// Renderiza dashboard
	output := model.renderDashboard()

	// Verifica que contém elementos esperados
	if len(output) == 0 {
		t.Error("Expected non-empty dashboard output")
	}

	// Verifica headers/tabs
	if !contains(output, "Dashboard") {
		t.Error("Expected dashboard output to contain 'Dashboard'")
	}
}

func TestRenderAlerts(t *testing.T) {
	model := New()
	model.width = 120
	model.height = 40
	model.ready = true

	// Adiciona anomalia
	anomaly := analyzer.Anomaly{
		Type:      analyzer.AnomalyTypeCPUSpike,
		Severity:  models.SeverityCritical,
		Cluster:   "test-cluster",
		Namespace: "default",
		HPAName:   "test-hpa",
		Message:   "CPU spike detected",
	}
	model.handleAnomaly(anomaly)

	// Renderiza alerts
	output := model.renderAlerts()

	if len(output) == 0 {
		t.Error("Expected non-empty alerts output")
	}

	if !contains(output, "Alertas") {
		t.Error("Expected alerts output to contain 'Alertas'")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long text", 10, "this is..."},
		{"exactly10!", 10, "exactly10!"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestWrapText(t *testing.T) {
	text := "this is a very long text that should be wrapped into multiple lines"
	result := wrapText(text, 20)

	// wrapText adiciona \n para quebrar linhas
	hasNewline := false
	for i := 0; i < len(result); i++ {
		if result[i] == '\n' {
			hasNewline = true
			break
		}
	}

	if !hasNewline {
		t.Error("Expected wrapped text to contain newlines")
	}
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
