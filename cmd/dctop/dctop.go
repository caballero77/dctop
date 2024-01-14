package main

import (
	"context"
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

	composeService, err := docker.NewComposeService(composeFilePath)
	if err != nil {
		fmt.Printf("error creating compose service: %v\n", err)
		slog.Error("error creating compose service: %v", err)
	}

	containersService, err := docker.NewContainersService(context.Background(), composeService.Stack())
	if err != nil {
		fmt.Printf("error creating docker service: %v\n", err)
		slog.Error("error creating docker service: %v", err)
	}
	defer containersService.Close()

	model, err := ui.NewUI(config, theme, containersService, composeService)
	if err != nil {
		fmt.Printf("error creating ui model: %v\n", err)
		slog.Error("error creating ui model: %v", err)
	}

	output := termenv.NewOutput(os.Stdout)
	backgroundColor := termenv.BackgroundColor()
	output.SetBackgroundColor(termenv.RGBColor("#2E3440"))

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(output))
	if _, err := p.Run(); err != nil {
		fmt.Printf("there's been an error: %v\n", err)
		slog.Error("there's been an error", "error", err)
		output.SetBackgroundColor(backgroundColor)
	}

	output.SetBackgroundColor(backgroundColor)
}
