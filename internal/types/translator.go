package types

import (
	"github.com/spf13/viper"
)

var targetLang = ""

func SetTranslatorLang(conf *viper.Viper) {
	targetLang = conf.GetString("translator.lang")
}

func GetTranslatorLang() string {
	return targetLang
}
