package cmd

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/google/go-github/v80/github"
	"slices"
	"strings"
)

func FetchLatestGitApi(owner, repo, prefix string) (string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)

	// Get the latest release
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Search assets by prefix
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.GetName(), prefix) {
			return asset.GetBrowserDownloadURL(), nil
		}
	}

	return "", fmt.Errorf("no asset found with prefix %q in release %s", prefix, release.GetTagName())
}

func UIManager() {
	var action string
	var rolling []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("task").
				Options(huh.NewOptions("Install", "Uninstall")...).
				Title("Choose an action").
				Description("This will make changes to your Bedrock").
				Value(&action),
			huh.NewMultiSelect[string]().
				Key("dll").
				Options(
					huh.NewOption("alteik/Modloader", "modloader").Selected(true),
					huh.NewOption("alteik/MinecraftForFree", "mcforfree").Selected(true),
				).
				Title("Choose to take on").
				Description("Those are thrid party from github, any risk is your responsible.").
				Value(&rolling),
		),
	).
		WithWidth(45).
		WithShowHelp(true).
		WithShowErrors(false)
	err := form.Run()
	if err != nil {
		logger.Fatal(err)
	}
	switch action {
	case "Install":
		if slices.Contains(rolling, "modloader") {
			url, err := FetchLatestGitApi("alteik", "Modloader", ".dll")
			if err != nil {
				logger.Error("Failed to fetch Modloader DLL", "err", err)
			} else {
				logger.Debug("Download URL", "url", url)
				logger.Info("Getting: alteik/Modloader")
			}
		}
		if slices.Contains(rolling, "mcforfree") {
			url, err := FetchLatestGitApi("alteik", "MinecraftForFree", ".dll")
			if err != nil {
				logger.Error("Failed to fetch Modloader DLL", "err", err)
			} else {
				logger.Debug("Download URL", "url", url)
				logger.Info("Getting: alteik/MinecraftForFree")
			}
		}
		logger.Info("didnt added yet.")
	case "UnInstall":
		logger.Warn("This will changes the .dll to .bak")
	}
}
