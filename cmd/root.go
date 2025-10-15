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
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bloodmagesoftware/climage/config"
	"github.com/bloodmagesoftware/climage/providers"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "climage",
	Short: "Generate images from text prompts using AI",
	Long:  `CLImage is a command-line tool for generating images from text prompts using various AI providers. Run without arguments to start an interactive session where you can enter prompts, switch models, adjust settings, and view generated images.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		errExit := errors.New("exit")

		prompt := ""
		lastPrompt := ""
		model := cfg.DefaultModel
		var modelSettings providers.ModelSettings

		// check if default model is valid
		{
			found := false
			for modelName, pm := range cfg.GetModels() {
				if model == modelName {
					found = true
					modelSettings = pm.Settings
					break
				}
			}
			// if not found, set to first provider's default model
			if !found {
				model = ""
				for modelName, pm := range cfg.GetModels() {
					model = modelName
					modelSettings = pm.Settings
					break
				}
			}
		}
		if model == "" {
			return fmt.Errorf("no model is available")
		}

		run := func() error {
			if err := huh.NewForm(huh.NewGroup(
				huh.NewText().
					Title("Prompt").
					Description("Enter your prompt for " + model + ".").
					Validate(huh.ValidateNotEmpty()).
					Value(&prompt),
			)).Run(); err != nil {
				return fmt.Errorf("failed to run prompt form: %w", err)
			}

			switch prompt {
			case "/models":
				var modelOptions []huh.Option[string]
				for modelName, model := range cfg.GetModels() {
					modelOptions = append(modelOptions, huh.NewOption(model.DisplayName, modelName))
				}
				if err := huh.NewForm(huh.NewGroup(
					huh.NewSelect[string]().
						Title("Model").
						Description("Select a model to generate an image with.").
						Options(modelOptions...).
						Value(&model),
				)).Run(); err != nil {
					return fmt.Errorf("failed to run model form: %w", err)
				}
				// update model settings
				for modelName, model := range cfg.GetModels() {
					if model.Name == modelName {
						modelSettings = model.Settings
						break
					}
				}

			case "/settings":
				if err := huh.NewForm(modelSettings.HuhGroup()).Run(); err != nil {
					return fmt.Errorf("failed to run settings form: %w", err)
				}

			case "/exit":
				return errExit

			case "/retry":
				prompt = lastPrompt
				fallthrough

			default:
				if strings.HasPrefix(prompt, "/") {
					fmt.Printf("invalid command: %q\n", prompt)
					break
				}
				modelParts := strings.SplitN(model, "/", 2)
				if len(modelParts) != 2 {
					return fmt.Errorf("invalid model: %q", model)
				}
				providerName := modelParts[0]
				modelName := modelParts[1]

				pp, err := providers.GetProviderByName(providerName)
				if err != nil {
					return fmt.Errorf("failed to get provider: %w", err)
				}
				out, err := pp.GenerateImage(cmd.Context(), modelName, prompt, modelSettings)
				if err != nil {
					return fmt.Errorf("failed to generate image: %w", err)
				}
				lastPrompt = prompt
				log.Println(prompt)
				for _, filePath := range out {
					fmt.Println(filePath)
					b := bounds(filePath)
					var cmd *exec.Cmd
					width := b.Dx()
					height := b.Dy()
					if width > height {
						cmd = exec.Command("viu", "--width", "80", filePath)
					} else {
						cmd = exec.Command("viu", "--height", "25", filePath)
					}
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					_ = cmd.Run()
				}
			}

			return nil
		}

		for {
			if err := run(); err != nil {
				if errors.Is(err, errExit) {
					break
				}
				// is huh user abort
				if errors.Is(err, huh.ErrUserAborted) {
					break
				}
				return err
			}
			prompt = ""
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SilenceUsage = true
}

func aspectRatio(imageFilePath string) float64 {
	b := bounds(imageFilePath)
	return float64(b.Dx()) / float64(b.Dy())
}

func bounds(imageFilePath string) image.Rectangle {
	f, err := os.Open(imageFilePath)
	if err != nil {
		return image.Rectangle{}
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return image.Rectangle{}
	}
	return img.Bounds()
}
