package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port       uint16      `yaml:"port"`
	Whitelist  []uuid.UUID `yaml:"whitelist"`
	InstanceID string      `yaml:"instance_id"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := Config{ // Default config
		Port:      25565,
		Whitelist: make([]uuid.UUID, 0),
	}
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	ApplyEnvOverrides(&cfg)
	return &cfg, nil
}

func ApplyEnvOverrides(cfg *Config) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found")
	}

	instanceid := os.Getenv("INSTANCE_ID")
	if instanceid != "" {
		cfg.InstanceID = instanceid
	}

	whitelist := os.Getenv("WHITELIST")
	if whitelist != "" {
		cfg.Whitelist = loadWhitelist(whitelist)
	}
}

func loadWhitelist(val string) []uuid.UUID {
	split := strings.Split(val, ",")
	uuids := make([]uuid.UUID, len(split))

	for i := range split {
		uuids[i] = uuid.MustParse(strings.TrimSpace(split[i]))
	}
	return uuids
}
