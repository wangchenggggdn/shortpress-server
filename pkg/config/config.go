package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func NewConfig(p string) *viper.Viper {
	envConf := os.Getenv("APP_CONF")
	if envConf == "" {
		envConf = p
	}
	fmt.Println("load conf file:", envConf)
	return getConfig(envConf)
}

func getConfig(path string) *viper.Viper {
	conf := viper.New()
	// Read raw file so we can expand ${ENV_VAR} placeholders before viper parses
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	// Expand ${VAR} style placeholders using current environment
	expanded := os.ExpandEnv(string(raw))
	conf.SetConfigType("yaml")
	if err := conf.ReadConfig(strings.NewReader(expanded)); err != nil {
		panic(err)
	}
	// Allow environment variables to override keys (e.g. DOMAIN_HOSTING overrides domain.hosting)
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	conf.AutomaticEnv()
	return conf
}
