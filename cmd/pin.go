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
	data, resolvedTag, err := FetchGitAssetsbyTag("bubbles-wow", "mcbe-gdk-unpack-archive", "Minecraft.Windows.exe", mcbe.Version)
	if err != nil {
		logger.Error("failed to fetch release by tag", "requested", mcbe.Version, "err", err)
	}

	// read current saved version and patched flag from config.ini
	cfgVer, cfgPatched, err := readConfig()
	if err != nil {
		logger.Error("reading config", err)
	}

	if versionsMatch(cfgVer, mcbe.Version) {
		logger.Info("Version matches config, skipping download", "version", mcbe.Version)
		// Only patch if file exists and is not already marked patched
		if cfgPatched {
			logger.Info("patched", "file", pack)
		} else {
			if _, err := os.Stat(pack); err == nil {
				if err := PatchFile(pack, true); err != nil {
					logger.Error("patch failed", "err", err)
				} else {
					logger.Info("patched", "file", pack)
					// mark patched in config
					if err := writeConfig(cfgVer, true); err != nil {
						logger.Error("failed to write config", err)
					}
				}
			} else {
				logger.Warn("main.exe not found, skipping patch", "path", pack)
			}
		}
	} else {
		if err != nil || data == "" {
			// If we couldn't fetch and an existing main.exe is present, run it offline
			if _, statErr := os.Stat(pack); statErr == nil {
				logger.Warn("offline or no asset; running existing main.exe", "path", pack)
				// don't attempt to download or patch
			} else {
				logger.Error("no downloadable asset found for requested or variant tags", "requested", mcbe.Version)
			}
		} else {
			logger.Info("Downloading latest bedrock", "url", data, "dest", pack)
			if err := GetDownload(data, pack); err != nil {
				logger.Error("download failed", err)
				// If download failed but main.exe exists, run it
				if _, statErr := os.Stat(pack); statErr == nil {
					logger.Warn("download failed, running existing main.exe", "path", pack)
				}
			} else {
				// prefer the resolved tag if available, otherwise fall back to requested
				tagToWrite := mcbe.Version
				if resolvedTag != "" {
					tagToWrite = resolvedTag
				}
				// write version even if patch fails
				if err := writeConfig(tagToWrite, false); err != nil {
					logger.Error("failed to write config", err)
				}
				// Attempt to patch the downloaded main.exe
				if _, err := os.Stat(pack); err == nil {
					if err := PatchFile(pack, true); err != nil {
						logger.Error("patch failed", "err", err)
					} else {
						logger.Info("patched", "file", pack)
						if err := writeConfig(tagToWrite, true); err != nil {
							logger.Error("failed to write config", err)
						}
					}
				} else {
					logger.Warn("downloaded file not found after download", "path", pack)
				}
			}
		}
	}

	logger.Info("Launching Minecraft")
	execCmd := exec.Command(pack)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Start(); err != nil {
		logger.Fatal("Failed to launch Minecraft", "err", err)
	}
	logger.Info("Minecraft started")
}

// readConfigVersion reads ./config.ini and returns the value for a "version=" key if present
// readConfig reads ./config.ini and returns the version and whether it's patched
func readConfig() (string, bool, error) {
	b, err := os.ReadFile("config.ini")
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	s := string(b)
	var ver string
	var patched bool
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "version=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				ver = strings.TrimSpace(parts[1])
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "patched=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(strings.ToLower(parts[1]))
				if val == "1" || val == "true" || val == "yes" {
					patched = true
				}
			}
		}
	}
	return ver, patched, nil
}

// writeConfigVersion writes a simple config with the version key
// writeConfig writes version and patched flag to config.ini
func writeConfig(v string, patched bool) error {
	p := "0"
	if patched {
		p = "1"
	}
	content := fmt.Sprintf("version=%s\npatched=%s\n", v, p)
	return os.WriteFile("config.ini", []byte(content), 0o644)
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
