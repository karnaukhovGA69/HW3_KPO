package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env            string        `yaml:"env" env:"ENV" env-default:"local"`
	StoragePath    string        `yaml:"storage_path" env:"STORAGE_PATH" env-default:"./storage"`
	HTTPServer     HTTPServer    `yaml:"http_server"`
	AnalysisServer HTTPServer    `yaml:"analysis_server"`
	StorageDB      StorageDB     `yaml:"storage_db"`
	AnalysisDB     AnalysisDB    `yaml:"analysis_db"`
	Gateway        GatewayConfig `yaml:"gateway"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env:"HTTP_SERVER_ADDRESS" env-default:"0.0.0.0:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"120s"`
}

type StorageDB struct {
	DSN string `yaml:"dsn" env:"STORAGE_DB_DCN"`
}

type AnalysisDB struct {
	DSN string `yaml:"dsn" env:"ANALYSIS_DB_DSN"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/local.yaml"
	}

	var config Config
	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		log.Fatalf("cannot read config %s: %v", configPath, err)
	}
	return &config
}

type GatewayConfig struct {
	StorageBaseURL  string `yaml:"storage_base_url" env:"STORAGE_BASE_URL"`
	AnalysisBaseURL string `yaml:"analysis_base_url" env:"ANALYSIS_BASE_URL"`
	Address         string `yaml:"address" env:"GATEWAY_ADDRESS" env-default:"localhost:8052"`
}
