package main

import (
	"fmt"
	"os"

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

	// TODO: Após setup concluído, iniciar scan engine baseado na config
	// Isso será integrado quando o usuário confirmar a configuração

	// Executa TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao executar HPA Watchdog: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("HPA Watchdog finalizado")
}
