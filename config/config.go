/*
CLImage is a AI image generation CLI tool.
Copyright (C) 2025  Mayer & Ott GbR

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public Licen
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

func getConfigFilePath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(userConfigDir, "climage", "config.json"), nil
}

type Config struct {
	Providers []Provider
}

type Provider struct {
	Name string
}

func GetConfig() (Config, error) {
	userConfigPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get config file path: %w", err)
	}
	_ = os.MkdirAll(filepath.Dir(userConfigPath), 0755)
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return Config{}, nil
	}
	f, err := os.Open(userConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()
	var config Config
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	return config, nil
}

func (c Config) Save() error {
	userConfigPath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}
	_ = os.MkdirAll(filepath.Dir(userConfigPath), 0755)
	f, err := os.Create(userConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(c)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

const keyringServiceName = "climage"

func SetProviderAPIKey(providerName string, apiKey string) error {
	return keyring.Set(keyringServiceName, providerName, apiKey)
}

func GetProviderAPIKey(providerName string) (string, error) {
	return keyring.Get(keyringServiceName, providerName)
}

func DeleteProviderAPIKey(providerName string) error {
	return keyring.Delete(keyringServiceName, providerName)
}
