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

type ModelSettings []ModelSetting

type ModelSetting struct {
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

type Provider interface {
	GenerateImage(model string, prompt string, settings any) (string, error)
	GetModels() []string
	GetModelSettings(model string) []ModelSetting
	GetSettings() any
}

func GetProviders() []string {
	return []string{"Google"}
}
