package config

import "os"

type Config struct {
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

func CreateConfig() *Config {
	return &Config{
		Token:   getEnv("token", ""),
		Channel: getEnv("channel", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
