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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/auth/credentials"
	"github.com/zalando/go-keyring"
	"google.golang.org/genai"
)

var googleSettings = ModelSettings{
	{DisplayName: "Number of Images", Name: "number_of_images", Type: "int", DefaultValue: "1"},
	{DisplayName: "Aspect Ratio", Name: "aspect_ratio", Type: "enum:1:1|16:9|4:3|9:16|3:4", DefaultValue: "1:1"},
	{DisplayName: "Output Resolution", Name: "output_resolution", Type: "enum:1K|2K", DefaultValue: "1K"},
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
			Name:        "service_account_key",
			DisplayName: "Service Account Key File",
			Type:        "file",
			Secret:      false,
		},
		{
			Name:        "project_id",
			DisplayName: "Project ID",
			Type:        "string",
			Secret:      false,
		},
		{
			Name:        "location",
			DisplayName: "Location",
			Type:        "string",
			Secret:      false,
		},
	}
}

type googleCredentials struct {
	ProjectID string `json:"project_id"`
	Location  string `json:"location"`
}

func (p *GoogleProvider) SaveCredentials(credentials map[string]string) error {
	serviceAccountKeyB64, ok := credentials["service_account_key"]
	if !ok {
		return fmt.Errorf("service_account_key not provided")
	}
	serviceAccountKey, err := base64.StdEncoding.DecodeString(serviceAccountKeyB64)
	if err != nil {
		return fmt.Errorf("failed to decode service account key: %w", err)
	}
	projectID, ok := credentials["project_id"]
	if !ok {
		return fmt.Errorf("project_id not provided")
	}
	location, ok := credentials["location"]
	if !ok {
		return fmt.Errorf("location not provided")
	}

	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data dir: %w", err)
	}
	credentialDir := filepath.Join(dataDir, "google")
	if err := os.MkdirAll(credentialDir, 0700); err != nil {
		return fmt.Errorf("failed to create credential dir: %w", err)
	}
	credentialFile := filepath.Join(credentialDir, "service_account_key")
	if err := os.WriteFile(credentialFile, serviceAccountKey, 0600); err != nil {
		return fmt.Errorf("failed to write service account key: %w", err)
	}

	creds := googleCredentials{
		ProjectID: projectID,
		Location:  location,
	}
	encoded, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return keyring.Set(keyringServiceName, "google", string(encoded))
}

func (p *GoogleProvider) LoadCredentials() (map[string]string, error) {
	stored, err := keyring.Get(keyringServiceName, "google")
	if err != nil {
		return nil, fmt.Errorf("not logged in to Google")
	}

	var creds googleCredentials
	if err := json.Unmarshal([]byte(stored), &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	dataDir, err := getDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data dir: %w", err)
	}
	credentialFile := filepath.Join(dataDir, "google", "service_account_key")
	serviceAccountKey, err := os.ReadFile(credentialFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account key: %w", err)
	}

	return map[string]string{
		"service_account_key": base64.StdEncoding.EncodeToString(serviceAccountKey),
		"project_id":          creds.ProjectID,
		"location":            creds.Location,
	}, nil
}

func (p *GoogleProvider) DeleteCredentials() error {
	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data dir: %w", err)
	}
	credentialDir := filepath.Join(dataDir, "google")
	if err := os.RemoveAll(credentialDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credential dir: %w", err)
	}
	return keyring.Delete(keyringServiceName, "google")
}

func (p *GoogleProvider) Login(ctx context.Context, creds map[string]string) error {
	if p.client != nil {
		return nil
	}
	serviceAccountKeyB64, ok := creds["service_account_key"]
	if !ok {
		return fmt.Errorf("service_account_key not provided")
	}
	serviceAccountKey, err := base64.StdEncoding.DecodeString(serviceAccountKeyB64)
	if err != nil {
		return fmt.Errorf("failed to decode service account key: %w", err)
	}
	projectID, ok := creds["project_id"]
	if !ok {
		return fmt.Errorf("project_id not provided")
	}
	location, ok := creds["location"]
	if !ok {
		return fmt.Errorf("location not provided")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	authCreds, err := credentials.DetectDefault(&credentials.DetectOptions{
		CredentialsJSON: serviceAccountKey,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		return fmt.Errorf("failed to detect credentials: %w", err)
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:     projectID,
		Location:    location,
		Backend:     genai.BackendVertexAI,
		Credentials: authCreds,
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	resp, err := p.client.Models.GenerateImages(ctx, model, prompt, &genai.GenerateImagesConfig{
		NumberOfImages:   int32(GetModelSettingInt(settings, "number_of_images", 1)),
		AspectRatio:      GetModelSettingString(settings, "aspect_ratio", "1:1"),
		ImageSize:        GetModelSettingString(settings, "output_resolution", "1K"),
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
		if len(img.RAIFilteredReason) > 0 {
			fmt.Printf("RAI Filtered: %s\n", img.RAIFilteredReason)
		}
		if img.Image == nil || len(img.Image.ImageBytes) == 0 {
			continue
		}
		var ext string
		if len(img.Image.MIMEType) == 0 {
			img.Image.MIMEType = http.DetectContentType(img.Image.ImageBytes)
		}
		switch img.Image.MIMEType {
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpg"
		case "image/gif":
			ext = ".gif"
		default:
			if img.Image.MIMEType == "text/plain; charset=utf-8" {
				log.Printf("Text outout: %s", string(img.Image.ImageBytes))
			}
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
