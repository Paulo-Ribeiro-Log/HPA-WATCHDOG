package main

import (
	"fmt"
	"os"

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

	// Goroutine para monitorar confirmação do setup e iniciar scan
	go func() {
		for {
			// Aguarda 1 segundo
			select {
			case <-model.GetSetupDoneChan():
				config := model.GetScanConfig()
				if config == nil {
					log.Warn().Msg("Configuração de scan não disponível")
					continue
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
				)

				if err := scanEngine.Start(); err != nil {
					log.Error().Err(err).Msg("Erro ao iniciar scan engine")
					continue
				}

				// Atualiza estado do model
				model.SetScanRunning(true)

				// Aguarda comandos de pausa/stop
				for {
					select {
					case <-model.GetPauseChan():
						if scanEngine.IsPaused() {
							scanEngine.Resume()
							model.SetScanPaused(false)
						} else {
							scanEngine.Pause()
							model.SetScanPaused(true)
						}

					case <-model.GetStopChan():
						log.Info().Msg("Parando scan engine")
						scanEngine.Stop()
						model.SetScanRunning(false)
						return
					}
				}
			}
		}
	}()

	// Executa TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao executar HPA Watchdog: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("HPA Watchdog finalizado")
}
