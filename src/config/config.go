package config

import (
"encoding/json"
"os"
"path/filepath"
)

const DefaultFaro = "190.220.45.26:54321"

type Config struct {
Faro string `json:"faro"`
}

func GetConfigPath() string {
home, err := os.UserHomeDir()
if err != nil {
return ".xion/config.json"
}
return filepath.Join(home, ".xion", "config.json")
}

func LoadConfig() (*Config, error) {
configPath := GetConfigPath()

data, err := os.ReadFile(configPath)
if err != nil {
return &Config{Faro: DefaultFaro}, nil
}

var cfg Config
if err := json.Unmarshal(data, &cfg); err != nil {
return nil, err
}

if cfg.Faro == "" {
cfg.Faro = DefaultFaro
}

return &cfg, nil
}

func SaveConfig(cfg *Config) error {
configPath := GetConfigPath()

dir := filepath.Dir(configPath)
if err := os.MkdirAll(dir, 0755); err != nil {
return err
}

data, err := json.MarshalIndent(cfg, "", "  ")
if err != nil {
return err
}

return os.WriteFile(configPath, data, 0644)
}

func GetFaroAddr() string {
cfg, err := LoadConfig()
if err == nil && cfg.Faro != "" {
return cfg.Faro
}

if faro := os.Getenv("XION_FARO"); faro != "" {
return faro
}

return DefaultFaro
}

func SetFaroAddr(addr string) error {
cfg, err := LoadConfig()
if err != nil {
cfg = &Config{}
}

cfg.Faro = addr
return SaveConfig(cfg)
}
