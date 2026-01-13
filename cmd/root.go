/*
Copyright © 2025 @Veha0001
*/
package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const (
	ActionRun     = "run"
	ActionPatch   = "patch"
	ActionRestore = "restore"
	ActionExit    = "exit"
)

var Version string = "devel"
var (
	cmd_opt_play  bool
	cmd_opt_patch bool
)
var logger = log.New(os.Stderr)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "bedmikun",
	Short:   "A program provide no trial on bedrock.",
	Long:    "A penguin trying to break bedrock, with a diamond pickaxe.",
	Run:     runBedmikun,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if len(os.Args) == 1 {
		runBedmikun(rootCmd, []string{})
		os.Exit(0)
	}
	err := fang.Execute(context.Background(), rootCmd)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("Bedmikun, version: " + Version)
	rootCmd.PersistentFlags().BoolVarP(&cmd_opt_play, "play", "g", false, "Play the game.")
	rootCmd.PersistentFlags().BoolVarP(&cmd_opt_patch, "patch", "w", false, "Patch the Minecraft.Windows.exe here.")
}
