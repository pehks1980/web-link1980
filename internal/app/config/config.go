package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config - конфиг структура по файлу .env
type Config struct {
	PORT string `envconfig:"PORT" required:"true"`
}

// New - cоздание конфига из файла .env
func New(cfgFile string) (*Config, error) {
	err := godotenv.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load config file %q %w", cfgFile, err)
	}

	config := new(Config)
	err = envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("get config from env variables: %w", err)
	}

	return config, nil
}
