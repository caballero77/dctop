package configuration

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

func NewConfiguration() (config *viper.Viper, theme Theme, err error) {
	config = viper.New()

	config.SetConfigName("config")
	config.SetConfigType("yaml")
	config.AddConfigPath("/usr/share/dctop/")
	err = config.ReadInConfig()
	if err != nil {
		return nil, theme, err
	}
	generalConfigDefaults(config)

	themeConfig := viper.New()
	themeName, ok := config.Get("theme").(string)
	if !ok {
		return nil, theme, errors.New("can't find theme name config")
	}

	themeConfig.SetConfigName("themes")
	themeConfig.SetConfigType("yaml")
	themeConfig.SetConfigFile(fmt.Sprintf("/usr/share/dctop/themes/%s.yaml", themeName))
	err = themeConfig.ReadInConfig()
	if err != nil {
		return nil, theme, err
	}

	theme = newTheme(themeConfig)

	return config, theme, nil
}
