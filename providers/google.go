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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
	"google.golang.org/genai"
)

var googleSettings = ModelSettings{
	{DisplayName: "Number of Images", Name: "number_of_images", Type: "int", DefaultValue: "1"},
	{DisplayName: "Aspect Ratio", Name: "aspect_ratio", Type: "enum:1:1|16:9|4:3|9:16|3:4", DefaultValue: "1:1"},
}

var GoogleModels = []Model{
	{Name: "imagen-4.0-generate-001", DisplayName: "Imagen 4", Settings: googleSettings},
	{Name: "imagen-4.0-ultra-generate-001", DisplayName: "Imagen 4 Ultra", Settings: googleSettings},
	{Name: "imagen-4.0-fast-generate-001", DisplayName: "Imagen 4 Fast", Settings: googleSettings},
}

type GoogleProvider struct {
	client *genai.Client
}

func init() {
	Providers = append(Providers, &GoogleProvider{})
}

func (p *GoogleProvider) GetName() string {
	return "google"
}

func (p *GoogleProvider) GetLoginFields() []LoginField {
	return []LoginField{
		{
			Name:        "api_key",
			DisplayName: "API Key",
			Type:        "string",
			Secret:      true,
		},
	}
}

func (p *GoogleProvider) SaveCredentials(credentials map[string]string) error {
	apiKey, ok := credentials["api_key"]
	if !ok {
		return fmt.Errorf("api_key not provided")
	}
	return keyring.Set(keyringServiceName, "google", apiKey)
}

func (p *GoogleProvider) LoadCredentials() (map[string]string, error) {
	apiKey, err := keyring.Get(keyringServiceName, "google")
	if err != nil {
		return nil, fmt.Errorf("not logged in to Google")
	}
	return map[string]string{"api_key": apiKey}, nil
}

func (p *GoogleProvider) DeleteCredentials() error {
	return keyring.Delete(keyringServiceName, "google")
}

func (p *GoogleProvider) Login(ctx context.Context, credentials map[string]string) error {
	if p.client != nil {
		return nil
	}
	apiKey, ok := credentials["api_key"]
	if !ok {
		return fmt.Errorf("api_key not provided")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create GenAI client: %w", err)
	}
	p.client = client
	return nil
}

func (p *GoogleProvider) Close() error {
	p.client = nil
	return nil
}

func (p *GoogleProvider) GenerateImage(ctx context.Context, model string, prompt string, settings ModelSettings) ([]string, error) {
	if p.client == nil {
		credentials, err := p.LoadCredentials()
		if err != nil {
			return nil, err
		}
		if err := p.Login(ctx, credentials); err != nil {
			return nil, fmt.Errorf("failed to login to Google: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	resp, err := p.client.Models.GenerateImages(ctx, model, prompt, &genai.GenerateImagesConfig{
		NumberOfImages:   int32(GetModelSettingInt(settings, "number_of_images", 1)),
		AspectRatio:      GetModelSettingString(settings, "aspect_ratio", "1:1"),
		IncludeRAIReason: true,
	})
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}

	dir, err := getOutDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get out dir: %w", err)
	}
	_ = os.MkdirAll(dir, 0755)
	nowDateTime := time.Now().Format(time.RFC3339)
	var filePaths []string
	for i, img := range resp.GeneratedImages {
		var ext string
		switch img.Image.MIMEType {
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpg"
		case "image/gif":
			ext = ".gif"
		default:
			return nil, fmt.Errorf("unsupported image type: %q", img.Image.MIMEType)
		}
		filePath := filepath.Join(dir, fmt.Sprintf("%s_%x_%s", nowDateTime, i, ext))
		if err := os.WriteFile(filePath, img.Image.ImageBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to write image: %w", err)
		}

		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}

func (p *GoogleProvider) GetModels() []Model {
	return GoogleModels
}

func (p *GoogleProvider) GetModelSettings(model string) []ModelSetting { return nil }

func (p *GoogleProvider) GetSettings() any {
	return nil
}
