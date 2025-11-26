/*
Copyright © 2025 @Veha0001
*/
package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const (
	ActionRun     = "run"
	ActionPatch   = "patch"
	ActionRestore = "restore"
	ActionExit    = "exit"
)

var logger = log.New(os.Stderr)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bedmikun",
	Short: "a program provide no trial on bedrock",
	Long:  `Bedmikun u`,
	Run:   runRootCmdUI, // Call the function moved to cmd/ui.go
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bedmikun.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}