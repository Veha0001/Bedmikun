package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// runRootCmdUI contains the main interactive UI logic for the bedmikun application.
func runRootCmdUI(cmd *cobra.Command, args []string) {
	logger.Info("Loading...")
	mcInfo, err := GetMinecraftInfo()
	if err != nil {
		logger.Fatal("Failed to get Minecraft info", "err", err)
	}

	// Debug: full path
	logger.Debug("Minecraft installed at", "path", mcInfo.InstallLocation)

	// Info: only version
	logger.Info("Minecraft installed", "version", mcInfo.Version)

	var action string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Action").
				Options(
					huh.NewOption("Run", ActionRun),
					huh.NewOption("Patch", ActionPatch),
					huh.NewOption("Restore Backup", ActionRestore),
					huh.NewOption("Exit", ActionExit),
				).
				Value(&action),
		),
	).Run()
	if err != nil {
		logger.Fatal("UI failed", "err", err)
	}

	switch action {
	case ActionRun:
		exePath := filepath.Join(mcInfo.InstallLocation, "Minecraft.Windows.exe")
		logger.Info("Launching Minecraft", "path", exePath)
		execCmd := exec.Command(exePath)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		if err := execCmd.Start(); err != nil {
			logger.Fatal("Failed to launch Minecraft", "err", err)
		}
		logger.Info("Minecraft started")

	case ActionPatch:
		var useDetectedPath bool
		autodetectedPath := filepath.Join(mcInfo.InstallLocation, "Minecraft.Windows.exe")
		path := autodetectedPath // Initialize path with the autodetected value

		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Use auto-detected path?").
					Description(autodetectedPath).
					Value(&useDetectedPath),
			),
		)
		if err := confirmForm.Run(); err != nil {
			logger.Fatal("UI failed", "err", err)
		}

		if !useDetectedPath { // If user says NO to detected path, then prompt for manual input
			inputForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter path to Minecraft.Windows.exe").
						Value(&path),
				),
			)
			if err := inputForm.Run(); err != nil {
				logger.Fatal("UI failed", "err", err)
			}
		}

		if path == "" {
			logger.Fatal("Path cannot be empty")
		}

		if err := PatchFile(path); err != nil {
			logger.Fatal("Failed to patch file", "err", err)
		}
		logger.Info("Patch complete")
		// Directory deletion functionality removed per user request.

	case ActionRestore:
		var useDetectedPath bool
		autodetectedPath := filepath.Join(mcInfo.InstallLocation, "Minecraft.Windows.exe")
		path := autodetectedPath // Initialize path with the autodetected value

		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Use auto-detected path for restore?").
					Description(autodetectedPath).
					Value(&useDetectedPath),
			),
		)
		if err := confirmForm.Run(); err != nil {
			logger.Fatal("UI failed", "err", err)
		}

		if !useDetectedPath {
			inputForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter path to Minecraft.Windows.exe to restore").
						Value(&path),
				),
			)
			if err := inputForm.Run(); err != nil {
				logger.Fatal("UI failed", "err", err)
			}
		}

		if path == "" {
			logger.Fatal("Path cannot be empty")
		}

		if err := RestoreFile(path); err != nil {
			logger.Fatal("Failed to restore file", "err", err)
		}
		logger.Info("Restore complete")

	case ActionExit:
		logger.Info("Exiting")
		return

	default:
		logger.Info("Unknown action")
	}
}
