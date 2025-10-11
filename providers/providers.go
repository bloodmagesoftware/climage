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

package providers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/bloodmagesoftware/climage/downloads"
	"github.com/charmbracelet/huh"
)

type Model struct {
	Name        string
	DisplayName string
	Settings    ModelSettings
}

type ModelSettings []*ModelSetting

func IsOfType(v string, t string) bool {
	switch t {
	case "int":
		_, err := strconv.Atoi(v)
		return err == nil
	case "float":
		_, err := strconv.ParseFloat(v, 64)
		return err == nil
	case "string":
		return true
	case "boolean":
		return v == "true" || v == "false"
	default:
		if union, ok := strings.CutPrefix(t, "enum:"); ok {
			unionFields := strings.Split(union, "|")
			if len(unionFields) == 0 {
				return false
			} else if len(unionFields) <= 8 {
				// select
				return slices.Contains(unionFields, v)
			} else {
				// text input
				return slices.Contains(unionFields, v)
			}
		} else {
			log.Printf("unknown model setting type: %q", t)
			return false
		}
	}
}

func (ms ModelSettings) HuhGroup() *huh.Group {
	var fields []huh.Field
	for _, m := range ms {
		if m.Value == "" && m.DefaultValue != "" {
			m.Value = m.DefaultValue
		}
		switch m.Type {
		case "int":
			fields = append(fields, huh.NewInput().
				Title(m.DisplayName).
				Validate(func(s string) error {
					_, err := strconv.Atoi(s)
					return err
				}).
				Value(&m.Value))
		case "float":
			fields = append(fields, huh.NewInput().
				Title(m.DisplayName).
				Validate(func(s string) error {
					_, err := strconv.ParseFloat(s, 64)
					return err
				}).
				Value(&m.Value))
		case "string":
			fields = append(fields, huh.NewInput().
				Title(m.DisplayName).
				Validate(huh.ValidateNotEmpty()).
				Value(&m.Value))
		case "boolean":
			fields = append(fields, huh.NewSelect[string]().
				Title(m.DisplayName).
				Options(huh.NewOptions("true", "false")...).
				Value(&m.Value))
		default:
			if union, ok := strings.CutPrefix(m.Type, "enum:"); ok {
				unionFields := strings.Split(union, "|")
				if len(unionFields) == 0 {
					continue
				} else if len(unionFields) <= 8 {
					// select
					options := make([]huh.Option[string], len(unionFields))
					for i, o := range unionFields {
						options[i] = huh.NewOption(o, o)
					}
					fields = append(fields, huh.NewSelect[string]().
						Title(m.DisplayName).
						Options(options...).
						Value(&m.Value))
				} else {
					// text input
					fields = append(fields, huh.NewInput().
						Title(m.DisplayName).
						Validate(func(s string) error {
							if slices.Contains(unionFields, s) {
								return nil
							}
							return fmt.Errorf("invalid value")
						}).
						Suggestions(unionFields).
						Value(&m.Value))
				}
			} else {
				log.Printf("unknown model setting type: %q", m.Type)
			}
		}
	}
	return huh.NewGroup(fields...)
}

func GetModelSettingString(ms ModelSettings, name string, defaultValue string) string {
	for _, m := range ms {
		if m.Name == name {
			log.Printf("setting %q to %q", name, m.Value)
			return m.Value
		}
	}
	log.Printf("setting %q to default %q", name, defaultValue)
	return defaultValue
}
func GetModelSettingBool(ms ModelSettings, name string, defaultValue bool) bool {
	for _, m := range ms {
		if m.Name == name {
			return m.Value == "true"
		}
	}
	return defaultValue
}
func GetModelSettingInt(ms ModelSettings, name string, defaultValue int) int {
	for _, m := range ms {
		if m.Name == name {
			v, err := strconv.Atoi(m.Value)
			if err != nil {
				return defaultValue
			}
			return v
		}
	}
	return defaultValue
}

type ModelSetting struct {
	DisplayName  string
	Name         string
	Type         string
	DefaultValue string
	Value        string
}

type ModelSettingType uint8

const (
	ModelSettingTypeString ModelSettingType = iota
	ModelSettingTypeNumber
	ModelSettingTypeBoolean
)

const keyringServiceName = "climage"

type LoginField struct {
	Name        string
	DisplayName string
	Type        string
	Secret      bool
}

type Provider interface {
	GetName() string
	GetLoginFields() []LoginField
	SaveCredentials(credentials map[string]string) error
	LoadCredentials() (map[string]string, error)
	DeleteCredentials() error
	Login(ctx context.Context, credentials map[string]string) error
	GenerateImage(ctx context.Context, model string, prompt string, settings ModelSettings) ([]string, error)
	GetModels() []Model
	GetModelSettings(model string) []ModelSetting
	GetSettings() any
	Close() error
}

var Providers []Provider

func GetProviderNames() []string {
	names := make([]string, len(Providers))
	for i, p := range Providers {
		names[i] = p.GetName()
	}
	return names
}

func GetProviderByName(name string) (Provider, error) {
	for _, p := range Providers {
		if p.GetName() == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", name)
}

func Close() error {
	var errs []error
	for _, p := range Providers {
		if err := p.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close provider %q: %w", p.GetName(), err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func getOutDir() (string, error) {
	dir, err := downloads.GetUserDownloadsDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user downloads dir: %w", err)
	}
	return filepath.Join(dir, "climage"), nil
}

func getDataDir() (string, error) {
	userDataDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(userDataDir, "climage", "data"), nil
}
