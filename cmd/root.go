/*
Copyright © 2025 @Veha0001
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"os"
)

const (
	ActionRun     = "run"
	ActionPatch   = "patch"
	ActionRestore = "restore"
	ActionExit    = "exit"
)

var Version string = "devel"
var (
	cmd_play     bool
	cmd_winpatch bool
)
var logger = log.New(os.Stderr)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bedmikun",
	Short: "A program provide no trial on bedrock.",
	Long:  "A penguin trying to break bedrock, with a diamond pickaxe.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Info("Loading...")
	},
	Run:     runBedmikun,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Run Actions on click
	if len(os.Args) == 1 {
		runBedmikun(rootCmd, []string{})
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		return
	}
	err := fang.Execute(context.Background(), rootCmd)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.SetVersionTemplate("Bedmikun, version: " + Version)
	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bedmikun.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&cmd_play, "play", "g", false, "Play the game.")
	rootCmd.PersistentFlags().BoolVarP(&cmd_winpatch, "patch", "w", false, "Patch the Minecraft.Windows.exe here.")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
