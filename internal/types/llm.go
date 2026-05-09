package types

import (
	"github.com/spf13/viper"
)

type LLMConfig struct {
	BaseURL string
	Model   string
	APIKey  string
}

var llmCfg = LLMConfig{}

func SetLLMConfig(conf *viper.Viper) {
	llmCfg.BaseURL = conf.GetString("llm.base_url")
	llmCfg.Model = conf.GetString("llm.model")
	llmCfg.APIKey = conf.GetString("llm.apikey")
}

func GetLLMConfig() LLMConfig {
	return llmCfg
}
