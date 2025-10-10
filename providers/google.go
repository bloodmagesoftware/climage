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
	"log"
	"time"

	"google.golang.org/genai"
)

type GoogleProvider struct {
	client *genai.Client
}

func NewGoogleProvider(apiKey string) (*GoogleProvider, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	return &GoogleProvider{
		client,
	}, nil
}

func (p *GoogleProvider) GenerateImage(model string, prompt string, settings any) (string, error) {
	return "", nil
}

func (p *GoogleProvider) GetModels() []string {
	var models []string

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for a, err := range p.client.Models.All(ctx) {
		if err != nil {
			log.Printf("failed to get models: %v", err)
			continue
		}
		models = append(models, a.Name)
	}
	return models
}

func (p *GoogleProvider) GetModelSettings(model string) []ModelSetting { return nil }

func (p *GoogleProvider) GetSettings() any {
	return nil
}
