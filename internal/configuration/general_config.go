package configuration

import "github.com/spf13/viper"

var (
	ContainersListHeightName = "containers_list_height"
	ProcessesListHeightName  = "processes_list_height"
	ThemeName                = "theme"
)

func generalConfigDefaults(config *viper.Viper) {
	config.SetDefault(ContainersListHeightName, 10)
	config.SetDefault(ProcessesListHeightName, 10)
	config.SetDefault(ThemeName, "nord")
}
