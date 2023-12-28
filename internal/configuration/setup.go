package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

func NewConfiguration() (config *viper.Viper, theme Theme, err error) {
	var path string
	if runtime.GOOS == "windows" {
		ex, err := os.Executable()
		if err != nil {
			return nil, theme, err
		}

		path = filepath.Dir(ex)
	} else {
		path = "/usr/share/dctop"
	}
	config = viper.New()

	config.SetConfigName("config")
	config.SetConfigType("yaml")
	config.AddConfigPath(path)
	err = config.ReadInConfig()
	if err != nil {
		if !errors.Is(err, viper.ConfigFileNotFoundError{}) {
			return nil, theme, err
		}
	}
	generalConfigDefaults(config)

	themeConfig := viper.New()
	themeName, ok := config.Get("theme").(string)
	if !ok {
		return nil, theme, errors.New("can't find theme name config")
	}

	themeConfig.SetConfigName("themes")
	themeConfig.SetConfigType("yaml")
	themeConfig.SetConfigFile(fmt.Sprintf("./%s/%s.yaml", filepath.Join(path, "themes"), themeName))
	err = themeConfig.ReadInConfig()
	if err != nil {
		if !errors.Is(err, viper.ConfigFileNotFoundError{}) {
			return nil, theme, err
		}
	}

	theme = newTheme(themeConfig)

	return config, theme, nil
}
