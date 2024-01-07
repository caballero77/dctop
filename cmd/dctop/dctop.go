package main

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func main() {
	file, err := os.OpenFile("logs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	logger := slog.New(slog.NewTextHandler(file, nil))
	slog.SetDefault(logger)

	composeFilePath := os.Args[1]

	config, theme, err := configuration.NewConfiguration()
	if err != nil {
		panic(err)
	}

	service, err := docker.ProvideComposeService(composeFilePath)
	if err != nil {
		panic(err)
	}
	model := ui.NewUI(config, theme, service)

	output := termenv.NewOutput(os.Stdout)
	backgroundColor := termenv.BackgroundColor()
	output.SetBackgroundColor(termenv.RGBColor("#2E3440"))

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(output))
	if _, err := p.Run(); err != nil {
		slog.Error("There's been an error", err)
		output.SetBackgroundColor(backgroundColor)
	}

	output.SetBackgroundColor(backgroundColor)
}
