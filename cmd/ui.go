package cmd

import (
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func runBedmikun(cmd *cobra.Command, args []string) {
	var action string
	if cmd_winpatch {
		if err := PatchFile("Minecraft.Windows.exe", false); err != nil {
			logger.Fatal("Failed to patch file", "err", err)
		}
		os.Exit(0)
	}
	if cmd_play {
		mcbe, err := GetMinecraftInfo()
		if err != nil {
			logger.Fatal("Failed to get Minecraft info", "err", err)
		} else if mcbe == nil {
			logger.Fatal("Minecraft is not installed or could not be found")
		}
		DRunBedrock(*mcbe)
		os.Exit(0)
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Action").
				Options(
					huh.NewOption("Run", ActionRun),
					huh.NewOption("Patch", ActionPatch),
					huh.NewOption("Restore", ActionRestore),
					huh.NewOption("Exit", ActionExit),
				).
				Value(&action),
		),
	).Run()
	if err != nil {
		logger.Fatal("UI failed", "err", err)
	}

	mcInfo, err := GetMinecraftInfo()
	if err != nil {
		logger.Fatal("Failed to get Minecraft info", "err", err)
	} else if mcInfo == nil {
		logger.Fatal("Minecraft is not installed or could not be found")
	}
	if mcInfo != nil {
		// Debug: full path
		logger.Debug("Minecraft installed at", "path", mcInfo.InstallLocation)
		// Info: only version
		logger.Info("Minecraft installed", "version", mcInfo.Version)
	}
	switch action {

	case ActionExit:
		logger.Info("Exiting...")
		return

	case ActionRun:
		DRunBedrock(*mcInfo)

	case ActionPatch:
		var useDetectedPath bool
		autodetectedPath := filepath.Join(mcInfo.InstallLocation, "Minecraft.Windows.exe")
		path := autodetectedPath // Initialize path with the autodetected value
		basename := filepath.Base(path)

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
						Value(&basename),
				),
			)
			if err := inputForm.Run(); err != nil {
				logger.Fatal("UI failed", "err", err)
			}
			path = basename
		}

		if path == "" {
			logger.Fatal("Path cannot be empty")
		}

		var createBackup bool
		backupConfirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Create a backup of the original file?").
					Value(&createBackup),
			),
		)

		if err := backupConfirmForm.Run(); err != nil {
			logger.Fatal("UI failed", "err", err)
		}

		if err := PatchFile(path, createBackup); err != nil {
			logger.Fatal("Failed to patch file", "err", err)
		}

	case ActionRestore:
		var useDetectedPath bool
		autodetectedPath := filepath.Join(mcInfo.InstallLocation, "Minecraft.Windows.exe")
		path := autodetectedPath // Initialize path with the autodetected value
		basename := filepath.Base(path)

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
						Value(&basename),
				),
			)
			if err := inputForm.Run(); err != nil {
				logger.Fatal("UI failed", "err", err)
			}
			path = basename
		}

		if path == "" {
			logger.Fatal("Path cannot be empty")
		}

		if err := RestoreFile(path); err != nil {
			logger.Fatal("Failed to restore file", "err", err)
		}

	default:
		logger.Info("Unknown action")
	}
}
