package configuration

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

func NewConfiguration() (config, theme *viper.Viper, err error) {
	config = viper.New()

	config.SetConfigName("config")
	config.SetConfigType("yaml")
	config.AddConfigPath("/usr/share/dctop/")
	err = config.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}
	generalConfigDefaults(config)

	theme = viper.New()
	themeName, ok := config.Get("theme").(string)
	if !ok {
		return nil, nil, errors.New("can't find theme name config")
	}

	theme.SetConfigName("themes")
	theme.SetConfigType("yaml")
	theme.SetConfigFile(fmt.Sprintf("/usr/share/dctop/themes/%s.yaml", themeName))
	err = theme.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}

	return config, theme, nil
}
