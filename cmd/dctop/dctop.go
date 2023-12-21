package main

import (
	"dctop/internal/configuration"
	"dctop/internal/docker"
	"dctop/internal/ui"
	"fmt"
	"net/http"
	"os"

	_ "net/http/pprof"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:8080", nil)
	}()

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
		fmt.Printf("Alas, there's been an error: %v", err)
		output.SetBackgroundColor(backgroundColor)
		os.Exit(1)
	}

	output.SetBackgroundColor(backgroundColor)
}
