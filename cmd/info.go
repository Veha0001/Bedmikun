package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
)

type McAppInfo struct {
	InstallLocation   string `json:"installLocation"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	PackageFamilyName string `json:"packageFamilyName"`
}

func GetMinecraftInfo() (*McAppInfo, error) {
	// Run PowerShell command with error handling and verbose output
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-AppxPackage -Name Microsoft.MinecraftUWP -ErrorAction Stop |
         Select-Object InstallLocation, Name, Version, PackageFamilyName |
         ConvertTo-Json -Depth 3`)

	// Capture both stdout and stderr
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell command: %w\nOutput: %s", err, string(out))
	}

	// Clean up the output (trim any unwanted whitespace)
	clean := strings.TrimSpace(string(out))
	if clean == "" || clean == "null" {
		var openStore bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Minecraft Bedrock not found.").
					Description("Would you like to open the Microsoft Store to install it?").
					Value(&openStore),
			),
		)
		if err := form.Run(); err == nil && openStore {
			// Open the Microsoft Store to install Minecraft
			_ = exec.Command("cmd", "/C", "start", "ms-windows-store://pdp/?productid=9NBLGGH2JHXJ").Run()
			return nil, fmt.Errorf("Minecraft not found, opening Microsoft Store for installation")
		}
		return nil, fmt.Errorf("Minecraft UWP not found. Please install it via the Microsoft Store")
	}

	// Debug: Output the raw content to check what's being returned
	logger.Debug("Raw PowerShell Output:\n", clean)

	// Try unmarshaling as an array first
	var arr []McAppInfo
	if err := json.Unmarshal([]byte(clean), &arr); err == nil {
		if len(arr) > 0 {
			return &arr[0], nil
		}
	}

	// If array parsing fails, try unmarshaling as a single object
	var info McAppInfo
	if err := json.Unmarshal([]byte(clean), &info); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nRaw output: %s", err, clean)
	}
	return &info, nil
}
