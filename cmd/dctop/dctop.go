package main

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui"
	"fmt"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"
)

func setupLogging(config *viper.Viper) func() {
	file, err := os.OpenFile("logs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(config.GetString("level"))); err != nil {
		level = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	return func() {
		err := file.Close()
		log.Fatalf("error closing file: %v", err)
	}
}

func main() {
	config, theme, err := configuration.NewConfiguration()
	if err != nil {
		log.Fatalf("error reading configuration: %v", err)
	}
	closeWriter := setupLogging(config.Sub("logs"))
	defer closeWriter()

	composeFilePath := os.Args[1]

	service, err := docker.ProvideComposeService(composeFilePath)
	if err != nil {
		fmt.Printf("error creating docker compose service: %v\n", err)
		slog.Error("error creating docker compose service: %v", err)
	}
	defer service.Close()

	model := ui.NewUI(config, theme, service)

	output := termenv.NewOutput(os.Stdout)
	backgroundColor := termenv.BackgroundColor()
	output.SetBackgroundColor(termenv.RGBColor("#2E3440"))

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(output))
	if _, err := p.Run(); err != nil {
		fmt.Printf("there's been an error: %v\n", err)
		slog.Error("there's been an error", "Error", err)
		output.SetBackgroundColor(backgroundColor)
	}

	output.SetBackgroundColor(backgroundColor)
}
