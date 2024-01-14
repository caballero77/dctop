package docker

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ComposeService struct {
	composePath string

	stack   string
	compose Compose
}

func NewComposeService(composePath string) (ComposeService, error) {
	stack, compose, err := getStack(composePath)
	if err != nil {
		var service ComposeService
		return service, fmt.Errorf("error reading stack name and compose file: %w", err)
	}

	return ComposeService{
		composePath: composePath,
		stack:       stack,
		compose:     compose,
	}, nil
}

func (service ComposeService) Stack() string { return service.stack }

func (service ComposeService) FilePath() string { return service.composePath }

func (service ComposeService) ComposeDown() error {
	slog.Debug("Executing down comand on compose file")

	cmd := exec.Command("docker-compose", "-f", service.composePath, "down") // #nosec G204
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error execution docker-compose down command: %w", err)
	}

	return nil
}

func (service ComposeService) ComposeUp() error {
	slog.Debug("Executing up comand on compose file")

	cmd := exec.Command("docker-compose", "-f", service.composePath, "up", "-d") // #nosec G204
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error execution docker-compose up command: %w", err)
	}

	return nil
}

func getStack(composePath string) (stack string, compose Compose, err error) {
	slog.Info("Start reading compose file")
	stack = filepath.Base(filepath.Dir(composePath))

	bytes, err := os.ReadFile(composePath)
	if err != nil {
		return "", compose, fmt.Errorf("error reading compose file: %w", err)
	}

	err = yaml.Unmarshal(bytes, &compose)
	if err != nil {
		return "", compose, fmt.Errorf("error unmarchaling compose file data: %w", err)
	}

	return stack, compose, nil
}
