package config

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct{}

func LoadConfig(configName string, paths ...string) (*Config, error) {
	var config = new(Config)

	v := viper.New()
	v.SetConfigName(configName)
	for _, path := range paths {
		v.AddConfigPath(path)
	}

	replacer := strings.NewReplacer(".", "_")
	v.SetEnvKeyReplacer(replacer)
	v.AutomaticEnv()

	// https://www.bookstack.cn/read/cobra/spilt.2.spilt.4.README.md
	//pflag.String("mediaDir", ".", "Directory to rename media in")
	//pflag.Parse()
	//err := viper.BindPFlags(pflag.CommandLine)
	//if err != nil {
	//	return nil, errors.Wrap(err, "BindPFlags")
	//}

	err := v.ReadInConfig()
	if err != nil {
		return nil, errors.Wrap(err, "reading config")
	}

	err = v.Unmarshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "v.Unmarshal()")
	}

	return config, nil
}
