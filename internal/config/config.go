package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type MikrotikConfig struct {
	Host string
	User string
	Pass string
}

type Config struct {
	Router1 MikrotikConfig
	Router2 MikrotikConfig
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Tidak menemukan file .env, menggunakan environment variable sistem")
	}

	return &Config{
		Router1: MikrotikConfig{
			Host: os.Getenv("MK1_HOST"),
			User: os.Getenv("MK1_USER"),
			Pass: os.Getenv("MK1_PASS"),
		},
		Router2: MikrotikConfig{
			Host: os.Getenv("MK2_HOST"),
			User: os.Getenv("MK2_USER"),
			Pass: os.Getenv("MK2_PASS"),
		},
	}
}
