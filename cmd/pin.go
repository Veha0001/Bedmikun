package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v80/github"
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
func FetchGitAssetsbyTag(owner, repo, prefix string, tag string) (string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)

	// Get the latest release
	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %q release: %s", tag, err)
	}

	// Search assets by prefix
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.GetName(), prefix) {
			return asset.GetBrowserDownloadURL(), nil
		}
	}

	return "", fmt.Errorf("no asset found with prefix %q in release %s", prefix, release.GetTagName())
}

func DRunBedrock(mcbe McAppInfo) {
	pack := filepath.Join(mcbe.InstallLocation, "main.exe")
	data, err := FetchGitAssetsbyTag("bubbles-wow", "mcbe-gdk-unpack-archive", "Minecraft.Windows.exe", mcbe.Version)
	if err != nil {
		logger.Error(err)
	}

	logger.Info("Launching Minecraft")
	execCmd := exec.Command("cmd", "/c", "start", pack)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Start(); err != nil {
		logger.Fatal("Failed to launch Minecraft", "err", err)
	}
	logger.Info("Minecraft started")
}
