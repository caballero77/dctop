package configuration

import "github.com/spf13/viper"

var (
	ContainersListHeigth = "containers_list_height"
	ProcessesListHeight  = "processes_list_height"
	Theme                = "theme"
)

func generalConfigDefaults(config *viper.Viper) {
	config.SetDefault(ContainersListHeigth, 10)
	config.SetDefault(ProcessesListHeight, 10)
	config.SetDefault(Theme, "nord")
}
