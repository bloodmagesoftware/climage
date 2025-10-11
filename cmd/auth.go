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

package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/bloodmagesoftware/climage/config"
	"github.com/bloodmagesoftware/climage/providers"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with image generation providers",
	Long:  `Manage authentication credentials for image generation providers. Use 'login' to add a new provider or 'logout' to remove an existing one.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to a provider",
	Long:  `Login to an image generation provider by providing your credentials. This allows CLImage to generate images using the selected provider.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var providerNames []string

		cfg, err := config.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

	full_provider_list:
		for _, a := range providers.GetProviderNames() {
			for _, b := range cfg.Providers {
				if a == b.Name {
					continue full_provider_list
				}
			}
			providerNames = append(providerNames, a)
		}

		if len(providerNames) == 0 {
			return fmt.Errorf("no providers are available")
		}

		providerName := providerNames[0]

		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Provider").
				Description("Select a provider to login with.").
				Options(huh.NewOptions(providerNames...)...).
				Validate(huh.ValidateNotEmpty()).
				Value(&providerName),
		)).Run(); err != nil {
			return fmt.Errorf("failed to run provider selection: %w", err)
		}

		provider, err := providers.GetProviderByName(providerName)
		if err != nil {
			return fmt.Errorf("failed to get provider: %w", err)
		}

		loginFields := provider.GetLoginFields()
		credentials := make(map[string]string)
		credentialValues := make(map[string]*string)
		var formFields []huh.Field

		for _, field := range loginFields {
			value := ""
			credentialValues[field.Name] = &value
			if field.Type == "file" {
				currentDirectory := "."
				if homeDir, err := os.UserHomeDir(); err == nil {
					currentDirectory = homeDir
				}
				formFields = append(formFields, huh.NewFilePicker().
					DirAllowed(false).
					ShowHidden(false).
					Title(field.DisplayName).
					Validate(huh.ValidateNotEmpty()).
					CurrentDirectory(currentDirectory).
					Value(credentialValues[field.Name]))
			} else {
				input := huh.NewInput().
					Title(field.DisplayName).
					Validate(huh.ValidateNotEmpty()).
					Value(credentialValues[field.Name])
				if field.Secret {
					input = input.EchoMode(huh.EchoModePassword)
				}
				formFields = append(formFields, input)
			}
		}

		if err := huh.NewForm(huh.NewGroup(formFields...)).Run(); err != nil {
			return fmt.Errorf("failed to run login form: %w", err)
		}

		for _, field := range loginFields {
			if field.Type == "file" && credentialValues[field.Name] != nil {
				// read file contents
				f, err := os.Open(*credentialValues[field.Name])
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				b, err := io.ReadAll(f)
				_ = f.Close()
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				b64 := base64.StdEncoding.EncodeToString(b)
				credentialValues[field.Name] = &b64
			}
		}

		for name, valuePtr := range credentialValues {
			credentials[name] = *valuePtr
		}

		if err := provider.Login(cmd.Context(), credentials); err != nil {
			return fmt.Errorf("failed to login with provided credentials: %w", err)
		}

		if err := provider.SaveCredentials(credentials); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		cfg.Providers = append(cfg.Providers, config.Provider{
			providerName,
		})
		if err = cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from a provider",
	Long:  `Logout from an image generation provider. This removes the provider's API key from your system and prevents CLImage from using it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		if len(cfg.Providers) == 0 {
			return fmt.Errorf("not logged in to any provider")
		}

		loggedInProviders := make([]string, len(cfg.Providers))
		for i, p := range cfg.Providers {
			loggedInProviders[i] = p.Name
		}

		providerName := loggedInProviders[0]
		if err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Provider").
				Description("Select a provider to logout from.").
				Options(huh.NewOptions(loggedInProviders...)...).
				Value(&providerName),
		)).Run(); err != nil {
			return fmt.Errorf("failed to run logout form: %w", err)
		}

		provider, err := providers.GetProviderByName(providerName)
		if err != nil {
			return fmt.Errorf("failed to get provider: %w", err)
		}

		if err := provider.DeleteCredentials(); err != nil {
			return fmt.Errorf("failed to delete credentials: %w", err)
		}

		newProviders := make([]config.Provider, 0, len(cfg.Providers)-1)
		for _, p := range cfg.Providers {
			if p.Name != providerName {
				newProviders = append(newProviders, p)
			}
		}
		cfg.Providers = newProviders

		if err = cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)

	rootCmd.AddCommand(authCmd)
}
