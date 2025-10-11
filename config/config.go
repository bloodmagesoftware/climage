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
	"iter"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bloodmagesoftware/climage/providers"
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
	Providers            []Provider        `json:"providers"`
	DefaultModel         string            `json:"default_model"`
	DefaultModelSettings map[string]string `json:"default_model_settings"`
}

type Provider struct {
	Name string `json:"name"`
}

func (p Provider) Get() (providers.Provider, error) {
	for _, pp := range providers.Providers {
		if pp.GetName() == p.Name {
			return pp, nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", p.Name)
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

	// apply user default model settings
	for _, m := range config.GetModels() {
		for _, s := range m.Settings {
			if v, ok := config.GetDefaultModelSetting(s.Name); ok && providers.IsOfType(v, s.Type) {
				s.Value = v
			}
		}
	}
	return config, nil
}

func (cfg Config) Save() error {
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
	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	err = enc.Encode(cfg)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func (cfg Config) GetDefaultModelSetting(setting string) (string, bool) {
	v, ok := cfg.DefaultModelSettings[setting]
	return v, ok
}
func (cfg Config) GetDefaultModelSettingInt(setting string) (int, bool) {
	v, ok := cfg.DefaultModelSettings[setting]
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return i, true
}
func (cfg Config) GetDefaultModelSettingBool(setting string) (bool, bool) {
	v, ok := cfg.DefaultModelSettings[setting]
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}
	return b, true
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

func (cfg Config) GetModels() iter.Seq2[string, providers.Model] {
	return func(yield func(string, providers.Model) bool) {
		for _, p := range cfg.Providers {
			pp, err := p.Get()
			if err != nil {
				return
			}
			for _, m := range pp.GetModels() {
				if !yield(pp.GetName()+"/"+m.Name, m) {
					return
				}
			}
		}
	}
}
