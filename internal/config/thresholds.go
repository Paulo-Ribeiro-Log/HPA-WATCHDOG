package config

import (
	"fmt"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	"github.com/rs/zerolog/log"
)

// ThresholdManager gerencia thresholds de forma thread-safe
type ThresholdManager struct {
	thresholds *models.Thresholds
}

// NewThresholdManager cria um novo manager
func NewThresholdManager(thresholds models.Thresholds) *ThresholdManager {
	return &ThresholdManager{
		thresholds: &thresholds,
	}
}

// Get retorna uma cópia dos thresholds atuais (thread-safe)
func (tm *ThresholdManager) Get() models.Thresholds {
	tm.thresholds.RLock()
	defer tm.thresholds.RUnlock()

	// Retorna cópia para evitar race conditions
	return *tm.thresholds
}

// UpdateCPUWarning atualiza CPU warning threshold
func (tm *ThresholdManager) UpdateCPUWarning(value int32) error {
	if value < 1 || value > 100 {
		return fmt.Errorf("cpu_warning_percent must be between 1 and 100")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	if value >= tm.thresholds.CPUCriticalPercent {
		return fmt.Errorf("cpu_warning_percent must be < cpu_critical_percent (%d)", tm.thresholds.CPUCriticalPercent)
	}

	tm.thresholds.CPUWarningPercent = value
	log.Info().Int32("value", value).Msg("CPU warning threshold updated")
	return nil
}

// UpdateCPUCritical atualiza CPU critical threshold
func (tm *ThresholdManager) UpdateCPUCritical(value int32) error {
	if value < 1 || value > 100 {
		return fmt.Errorf("cpu_critical_percent must be between 1 and 100")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	if value <= tm.thresholds.CPUWarningPercent {
		return fmt.Errorf("cpu_critical_percent must be > cpu_warning_percent (%d)", tm.thresholds.CPUWarningPercent)
	}

	tm.thresholds.CPUCriticalPercent = value
	log.Info().Int32("value", value).Msg("CPU critical threshold updated")
	return nil
}

// UpdateMemoryWarning atualiza Memory warning threshold
func (tm *ThresholdManager) UpdateMemoryWarning(value int32) error {
	if value < 1 || value > 100 {
		return fmt.Errorf("memory_warning_percent must be between 1 and 100")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	if value >= tm.thresholds.MemoryCriticalPercent {
		return fmt.Errorf("memory_warning_percent must be < memory_critical_percent (%d)", tm.thresholds.MemoryCriticalPercent)
	}

	tm.thresholds.MemoryWarningPercent = value
	log.Info().Int32("value", value).Msg("Memory warning threshold updated")
	return nil
}

// UpdateMemoryCritical atualiza Memory critical threshold
func (tm *ThresholdManager) UpdateMemoryCritical(value int32) error {
	if value < 1 || value > 100 {
		return fmt.Errorf("memory_critical_percent must be between 1 and 100")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	if value <= tm.thresholds.MemoryWarningPercent {
		return fmt.Errorf("memory_critical_percent must be > memory_warning_percent (%d)", tm.thresholds.MemoryWarningPercent)
	}

	tm.thresholds.MemoryCriticalPercent = value
	log.Info().Int32("value", value).Msg("Memory critical threshold updated")
	return nil
}

// UpdateReplicaDeltaPercent atualiza replica delta percent
func (tm *ThresholdManager) UpdateReplicaDeltaPercent(value float64) error {
	if value < 0 {
		return fmt.Errorf("replica_delta_percent must be >= 0")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.ReplicaDeltaPercent = value
	log.Info().Float64("value", value).Msg("Replica delta percent updated")
	return nil
}

// UpdateReplicaDeltaAbsolute atualiza replica delta absolute
func (tm *ThresholdManager) UpdateReplicaDeltaAbsolute(value int32) error {
	if value < 0 {
		return fmt.Errorf("replica_delta_absolute must be >= 0")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.ReplicaDeltaAbsolute = value
	log.Info().Int32("value", value).Msg("Replica delta absolute updated")
	return nil
}

// UpdateTargetDeviation atualiza target deviation percent
func (tm *ThresholdManager) UpdateTargetDeviation(value float64) error {
	if value < 0 {
		return fmt.Errorf("target_deviation_percent must be >= 0")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.TargetDeviationPercent = value
	log.Info().Float64("value", value).Msg("Target deviation updated")
	return nil
}

// UpdateScalingStuckMinutes atualiza scaling stuck minutes
func (tm *ThresholdManager) UpdateScalingStuckMinutes(value int) error {
	if value < 1 {
		return fmt.Errorf("scaling_stuck_minutes must be >= 1")
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.ScalingStuckMinutes = value
	log.Info().Int("value", value).Msg("Scaling stuck minutes updated")
	return nil
}

// ToggleConfigChangeAlert toggle alert on config change
func (tm *ThresholdManager) ToggleConfigChangeAlert(enabled bool) {
	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.AlertOnConfigChange = enabled
	log.Info().Bool("enabled", enabled).Msg("Alert on config change toggled")
}

// ToggleResourceChangeAlert toggle alert on resource change
func (tm *ThresholdManager) ToggleResourceChangeAlert(enabled bool) {
	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	tm.thresholds.AlertOnResourceChange = enabled
	log.Info().Bool("enabled", enabled).Msg("Alert on resource change toggled")
}

// UpdateAll atualiza todos os thresholds de uma vez (útil para reload config)
func (tm *ThresholdManager) UpdateAll(newThresholds models.Thresholds) error {
	// Valida antes de aplicar
	if err := validateThresholds(&newThresholds); err != nil {
		return err
	}

	tm.thresholds.Lock()
	defer tm.thresholds.Unlock()

	*tm.thresholds = newThresholds
	log.Info().Msg("All thresholds updated")
	return nil
}

// validateThresholds valida thresholds
func validateThresholds(t *models.Thresholds) error {
	if t.CPUWarningPercent < 1 || t.CPUWarningPercent > 100 {
		return fmt.Errorf("cpu_warning_percent must be between 1 and 100")
	}

	if t.CPUCriticalPercent < 1 || t.CPUCriticalPercent > 100 {
		return fmt.Errorf("cpu_critical_percent must be between 1 and 100")
	}

	if t.CPUCriticalPercent <= t.CPUWarningPercent {
		return fmt.Errorf("cpu_critical_percent must be > cpu_warning_percent")
	}

	if t.MemoryWarningPercent < 1 || t.MemoryWarningPercent > 100 {
		return fmt.Errorf("memory_warning_percent must be between 1 and 100")
	}

	if t.MemoryCriticalPercent < 1 || t.MemoryCriticalPercent > 100 {
		return fmt.Errorf("memory_critical_percent must be between 1 and 100")
	}

	if t.MemoryCriticalPercent <= t.MemoryWarningPercent {
		return fmt.Errorf("memory_critical_percent must be > memory_warning_percent")
	}

	if t.ReplicaDeltaPercent < 0 {
		return fmt.Errorf("replica_delta_percent must be >= 0")
	}

	if t.ReplicaDeltaAbsolute < 0 {
		return fmt.Errorf("replica_delta_absolute must be >= 0")
	}

	if t.ScalingStuckMinutes < 1 {
		return fmt.Errorf("scaling_stuck_minutes must be >= 1")
	}

	return nil
}

// String retorna uma representação legível dos thresholds
func (tm *ThresholdManager) String() string {
	t := tm.Get()
	return fmt.Sprintf(
		"CPU: %d%%/%d%% | Memory: %d%%/%d%% | Replica Δ: %.0f%%/±%d | Target Δ: %.0f%% | Stuck: %dm",
		t.CPUWarningPercent,
		t.CPUCriticalPercent,
		t.MemoryWarningPercent,
		t.MemoryCriticalPercent,
		t.ReplicaDeltaPercent,
		t.ReplicaDeltaAbsolute,
		t.TargetDeviationPercent,
		t.ScalingStuckMinutes,
	)
}
