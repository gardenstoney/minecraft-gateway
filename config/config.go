package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		Port      uint16
		Whitelist []uuid.UUID
		Server    Server
	}

	Server struct {
		InstanceId string
		Port       uint16
	}
)

func LoadConfig(path string) (cfg *Config, err error) {
	err = godotenv.Load()
	if err != nil {
		slog.Warn("No .env file found")
	}

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	cfg = &Config{ // Default config
		Port: 25565,
	}

	meta, err := toml.NewDecoder(file).Decode(cfg)
	if err != nil {
		return
	}

	if !meta.IsDefined("Server", "InstanceId") {
		err = errors.New("Server.InstanceId is not defined")
	}

	fmt.Println(cfg)

	return cfg, err
}
