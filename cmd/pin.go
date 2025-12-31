package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

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
func FetchGitAssetsbyTag(owner, repo, prefix string, tag string) (string, string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)
	// generate candidate tags to try (original, strip trailing .0, drop last segment, and without/with leading v)
	candidates := generateTagCandidates(tag)

	var lastErr error
	for _, t := range candidates {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, t)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch %q release: %w", t, err)
			continue
		}

		// Search assets by prefix
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.GetName(), prefix) {
				return asset.GetBrowserDownloadURL(), t, nil
			}
		}
		lastErr = fmt.Errorf("no asset found with prefix %q in release %s", prefix, release.GetTagName())
	}

	// If GitHub lookups failed or produced no asset, fall back to GdkLinks' urls.json
	// which maps many release tags / Windows release identifiers to direct download URLs.
	urlsJSON := "https://raw.githubusercontent.com/MinecraftBedrockArchiver/GdkLinks/refs/heads/master/urls.json"
	resp, err := http.Get(urlsJSON) // nolint:gosec
	if err != nil {
		if lastErr != nil {
			return "", "", lastErr
		}
		return "", "", fmt.Errorf("failed to fetch urls.json: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		if lastErr != nil {
			return "", "", lastErr
		}
		return "", "", fmt.Errorf("failed to fetch urls.json: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if lastErr != nil {
			return "", "", lastErr
		}
		return "", "", fmt.Errorf("failed to read urls.json: %w", err)
	}

	// The JSON schema is: { "release": { "<tag>": ["url1","url2"] }, "preview": {...} }
	var collection map[string]map[string][]string
	if err := json.Unmarshal(body, &collection); err != nil {
		if lastErr != nil {
			return "", "", lastErr
		}
		return "", "", fmt.Errorf("failed to parse urls.json: %w", err)
	}

	// Collect candidate JSON keys that look like the release tag we want
	var candidateTags []string
	for _, section := range collection {
		for k, list := range section {
			// If any URL contains the exact Windows package version (tag) use that JSON key
			for _, u := range list {
				if strings.Contains(u, tag) || strings.Contains(u, strings.TrimPrefix(tag, "v")) {
					candidateTags = append(candidateTags, k)
					break
				}
			}
			// also accept keys that match via our versionsMatch heuristic
			if versionsMatch(k, tag) || versionsMatch(tag, k) {
				candidateTags = append(candidateTags, k)
			}
		}
	}

	// Try each candidate tag by querying GitHub releases for that tag and finding
	// an asset with the requested prefix. Prefer the first successful match.
	for _, candidate := range candidateTags {
		// try GitHub release by this candidate tag
		rel, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, candidate)
		if err != nil {
			// record and continue
			lastErr = fmt.Errorf("failed to fetch %q release: %w", candidate, err)
			continue
		}
		for _, asset := range rel.Assets {
			if strings.HasSuffix(asset.GetName(), prefix) {
				return asset.GetBrowserDownloadURL(), candidate, nil
			}
		}
		// no asset found for this candidate tag; continue
		lastErr = fmt.Errorf("no asset found with prefix %q in release %s", prefix, rel.GetTagName())
	}

	if lastErr != nil {
		return "", "", lastErr
	}
	return "", "", fmt.Errorf("no candidates to try for tag %q", tag)
}

// generateTagCandidates returns a list of unique tag variants to try when fetching
// a release by tag. It includes variations that strip trailing ".0" components and
// drop the last dot-separated segment, and adds versions with/without a leading 'v'.
func generateTagCandidates(tag string) []string {
	seen := map[string]bool{}
	var out []string

	base := strings.TrimSpace(tag)
	if base == "" {
		return nil
	}

	variants := []string{base}

	// without leading v
	if strings.HasPrefix(base, "v") {
		variants = append(variants, strings.TrimPrefix(base, "v"))
	}

	// drop last segment
	dropped := dropLast(base)
	if dropped != base {
		variants = append(variants, dropped)
	}

	// combine with and without leading v
	for _, v := range variants {
		vs := strings.TrimSpace(v)
		if vs == "" {
			continue
		}
		// add as-is
		if !seen[vs] {
			seen[vs] = true
			out = append(out, vs)
		}
		// add with leading v if not present
		if !strings.HasPrefix(vs, "v") {
			v2 := "v" + vs
			if !seen[v2] {
				seen[v2] = true
				out = append(out, v2)
			}
		}
	}

	return out
}

func DRunBedrock(mcbe McAppInfo) {
	pack := filepath.Join(mcbe.InstallLocation, "main.exe")
	// If main.exe exists, ask the user whether to Play, Reinstall, or Cancel
	if _, err := os.Stat(pack); err == nil {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Found a patched Minecraft Bedrock.").
					Affirmative("Reinstall.").
					Negative("Play it!").
					Value(&confirm),
			),
		)
		if err := form.Run(); err != nil {
			logger.Fatal("UI failed", "err", err)
		}

		if confirm { // User selected "Reinstall"
			logger.Info("Reinstall selected; removing existing file...", "path", pack)
			if err := os.Remove(pack); err != nil {
				logger.Fatal("failed to remove existing file; continuing", "err", err)
				os.Exit(1)
			}
			// continue to download flow
		} else { // User selected "Play it!"
			logger.Info("Found existing main.exe, launching...", "path", pack)
			execCmd := exec.Command(pack)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			if err := execCmd.Start(); err != nil {
				logger.Fatal("Failed to launch Minecraft", "err", err)
			}
			logger.Info("Minecraft Bedrock started.")
			return
		}
	}

	// main.exe not found -> attempt to download
	data, _, err := FetchGitAssetsbyTag("bubbles-wow", "mcbe-gdk-unpack-archive", "Minecraft.Windows.exe", mcbe.Version)
	if err != nil || data == "" {
		logger.Error("no downloadable asset found for requested or variant tags", "requested", mcbe.Version, "err", err)
		return
	}

	logger.Info("Downloading Minecraft Bedrock.", "url", data, "dest", pack)
	if err := GetDownload(data, pack); err != nil {
		logger.Error("download failed", err)
		// If download failed but main.exe exists, run it
		if _, statErr := os.Stat(pack); statErr == nil {
			logger.Warn("download failed, running existing main.exe", "path", pack)
			execCmd := exec.Command(pack)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			if err := execCmd.Start(); err != nil {
				logger.Fatal("Failed to launch Minecraft", "err", err)
			}
			logger.Info("Minecraft Bedrock started.")
			return
		}
		return
	}

	// Attempt to patch the downloaded main.exe (backup disabled)
	if err := PatchFile(pack, false); err != nil {
		logger.Error("patch failed", "err", err)
		// still try to launch the downloaded file
	}

	logger.Info("Launching Minecraft Bedrock...")
	execCmd := exec.Command(pack)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Start(); err != nil {
		logger.Fatal("Failed to launch Minecraft", "err", err)
	}
	logger.Info("Minecraft Bedrock started.")
}

// versionsMatch compares two version strings and returns true when they should be
// considered the same even when trailing ".0" or a final build number differs.
func versionsMatch(a, b string) bool {
	a = strings.TrimSpace(strings.TrimPrefix(a, "v"))
	b = strings.TrimSpace(strings.TrimPrefix(b, "v"))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if dropLast(a) == dropLast(b) {
		return true
	}
	return false
}

func dropLast(v string) string {
	parts := strings.Split(v, ".")
	if len(parts) > 1 {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, ".")
}
