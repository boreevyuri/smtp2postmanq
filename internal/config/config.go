package config

import "github.com/spf13/viper"

func Load() *viper.Viper {
	v := viper.New()
	v.AddConfigPath("configs")
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	return v
}
