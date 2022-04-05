package config

import "github.com/spf13/viper"

func Load(cfgPath string) *viper.Viper {
	v := viper.New()
	v.AddConfigPath(cfgPath)
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	return v
}
