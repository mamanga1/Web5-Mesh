// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

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

// GetXionHome devuelve el directorio base de XionIA (Jaula de Faraday)
func GetXionHome() string {
if home := os.Getenv("XION_HOME"); home != "" {
return home
}
home, err := os.UserHomeDir()
if err != nil {
return ".xion"
}
return filepath.Join(home, ".xion")
}

// GetConfigPath devuelve la ruta al archivo config.json dentro de la Jaula
func GetConfigPath() string {
return filepath.Join(GetXionHome(), "config.json")
}

func LoadConfig() (*Config, error) {
configPath := GetConfigPath()

data, err := os.ReadFile(configPath)
if err != nil {
if os.IsNotExist(err) {
return &Config{Faro: DefaultFaro}, nil
}
return nil, err
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
if err := os.MkdirAll(dir, 0700); err != nil {
return err
}

data, err := json.MarshalIndent(cfg, "", "  ")
if err != nil {
return err
}

return os.WriteFile(configPath, data, 0600)
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

func ResetFaroAddr() error {
return SetFaroAddr(DefaultFaro)
}
