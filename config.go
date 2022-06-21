package main

import (
	"github.com/spf13/viper"
)

type Config struct {
	ListenTo  uint16
	ReflectTo []string
	LogDiff   float32
}

// todo filter out bodypart movement based on XY and stuff like that

func ReadConfig() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("json")

	// read from remote config the first time.
	err := v.ReadInConfig()

	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	// unmarshal config
	err = v.Unmarshal(&cfg)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}
