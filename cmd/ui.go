package cmd

import (
	"os"
	"path/filepath"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)
var style_b = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(1, 2).
    PaddingChar('·').
    Margin(1, 2).
    MarginChar('/').
    Width(18)

func runBedmikun(cmd *cobra.Command, args []string) {
	lipgloss.Println(style_b.Render("Hello, Bedmikun"))
	var (
		useDetectedPath bool
		action          string
	)
	if cmd_opt_patch {
		if err := PatchFile("Minecraft.Windows.exe", false); err != nil {
			logger.Fatal("Failed to patch file", "err", err)
		}
		os.Exit(0)
	}
	if cmd_opt_play {
		mcInfo, err := GetMinecraftInfo()
		if err != nil {
			logger.Fatal("Failed to get Minecraft info", "err", err)
		} else if mcInfo == nil {
			logger.Fatal("Minecraft is not installed or could not be found")
		}
		DRunBedrock(mcInfo.Version, mcInfo.InstallLocation)
		os.Exit(0)
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an Action").
				Options(
					huh.NewOption("Run", ActionRun),
					huh.NewOption("Patch", ActionPatch),
					huh.NewOption("Restore", ActionRestore),
					huh.NewOption("Exit", ActionExit),
				).
				Value(&action),
		).Title("Bedrock-unpaid patcher.").Description("Free selection to go."),
	).Run()
	if err != nil {
		logger.Fatal("UI failed", "err", err)
	}
	var mcInfo *McAppInfo
	mcbe, err := GetMinecraftInfo()
	if err != nil {
		logger.Fatal("Failed to get Minecraft info", "err", err)
	} else if mcbe == nil {
		logger.Fatal("Minecraft is not installed or could not be found")
	}
	mcInfo = mcbe

	switch action {

	case ActionExit:
		logger.Info("Exiting...")
		return

	case ActionRun:
		DRunBedrock(mcInfo.Version, mcInfo.InstallLocation)

	case ActionPatch:
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
