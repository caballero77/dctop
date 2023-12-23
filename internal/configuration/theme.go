package configuration

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type Theme struct {
	*viper.Viper
}

func newTheme(config *viper.Viper) Theme {
	return Theme{Viper: config}
}

func (theme Theme) Sub(path string) Theme {
	return newTheme(theme.Viper.Sub(path))
}

func (theme Theme) GetColor(path string) lipgloss.Color {
	return lipgloss.Color(theme.GetString(path))
}
