package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
)

type McAppInfo struct {
	InstallLocation   string `json:"install_location"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	PackageFamilyName string `json:"package_family_name"`
}

func GetMinecraftInfo() (*McAppInfo, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-AppxPackage -Name Microsoft.MinecraftUWP | Select-Object InstallLocation, Name, Version, PackageFamilyName | ConvertTo-Json -Depth 3`)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pwsh failed: %w", err)
	}
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
		err := form.Run()
		if err == nil && openStore {
			exec.Command("explorer", "ms-windows-store://pdp/?productid=9NBLGGH2JHXJ").Start()
			return nil, fmt.Errorf("opening Microsoft Store, please install and restart")
		}

		return nil, fmt.Errorf("minecraft uwp not found")
	}

	// Handle array or single object
	if strings.HasPrefix(clean, "[") {
		var arr []McAppInfo
		if err := json.Unmarshal([]byte(clean), &arr); err != nil {
			return nil, fmt.Errorf("json parse array: %w\nraw=%s", err, clean)
		}
		if len(arr) == 0 {
			return nil, fmt.Errorf("no Minecraft package found")
		}
		return &arr[0], nil
	}

	var info McAppInfo
	if err := json.Unmarshal([]byte(clean), &info); err != nil {
		return nil, fmt.Errorf("json parse object: %w\nraw=%s", err, clean)
	}
	return &info, nil
}
