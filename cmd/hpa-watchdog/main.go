package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/engine"
	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuração de logging (para arquivo, não interfere na TUI)
	logFile, err := os.OpenFile("/tmp/hpa-watchdog.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Logger = zerolog.New(logFile).With().Timestamp().Logger()
	}

	log.Info().Msg("Iniciando HPA Watchdog")

	// Cria model da TUI
	model := tui.New()

	// Inicia programa Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Goroutine para monitorar confirmação do setup e gerenciar scan engine
	go func() {
		// Aguarda confirmação do setup
		<-model.GetSetupDoneChan()

		// Loop principal que permite reinícios
		for {
			config := model.GetScanConfig()
			if config == nil {
				log.Warn().Msg("Configuração de scan não disponível")
				return
			}

			log.Info().
				Str("mode", config.Mode.String()).
				Int("targets", len(config.Targets)).
				Msg("Iniciando scan engine")

			// Cria e inicia scan engine
			scanEngine := engine.New(
				config,
				model.GetSnapshotChan(),
				model.GetAnomalyChan(),
				model.GetStressResultChan(),
			)

			if err := scanEngine.Start(); err != nil {
				log.Error().Err(err).Msg("Erro ao iniciar scan engine")
				return
			}

			// Atualiza estado do model via canal
			startTime := time.Now()
			model.UpdateScanStatus(true, false, startTime)

			// Aguarda comandos de pausa/stop/restart
			shouldRestart := false
		controlLoop:
			for {
				select {
				case <-model.GetPauseChan():
					if scanEngine.IsPaused() {
						log.Info().Msg("Retomando scan engine")
						scanEngine.Resume()
						model.UpdateScanStatus(true, false, startTime)
					} else {
						log.Info().Msg("Pausando scan engine")
						scanEngine.Pause()
						model.UpdateScanStatus(true, true, startTime)
					}

				case <-model.GetStopChan():
					log.Info().Msg("Parando scan engine")
					scanEngine.Stop()
					model.UpdateScanStatus(false, false, time.Time{})
					return

				case <-model.GetRestartChan():
					log.Info().Msg("Reiniciando scan engine")
					scanEngine.Stop()
					model.UpdateScanStatus(false, false, time.Time{})
					shouldRestart = true
					break controlLoop // Sai do loop de controle para reiniciar
				}
			}

			// Se não deve reiniciar, sai do loop principal
			if !shouldRestart {
				return
			}

			// Aguarda um momento antes de reiniciar
			time.Sleep(200 * time.Millisecond)
			log.Info().Msg("Iniciando novo ciclo de scan")
		}
	}()

	// Executa TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao executar HPA Watchdog: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("HPA Watchdog finalizado")
}
