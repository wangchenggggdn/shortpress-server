package types

import "github.com/spf13/viper"

type GrokConfig struct {
	BaseURL string
	Model   string
	APIKey  string
}

var grokCfg = GrokConfig{}

func SetGrokConfig(conf *viper.Viper) {
	grokCfg.BaseURL = conf.GetString("grok.base_url")
	grokCfg.Model = conf.GetString("grok.model")
	grokCfg.APIKey = conf.GetString("grok.apikey")
}

func GetGrokConfig() GrokConfig {
	return grokCfg
}
