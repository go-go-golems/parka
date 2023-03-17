package main

import (
	"github.com/go-go-golems/parka/cmd/parka/cmds"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "parka",
	Short: "parka keeps command-line applications warm and fuzzy by giving them a fluffy web API",
}

func init() {
	// Add viper, initLogger, helpSystem, all that jazz

	rootCmd.AddCommand(cmds.ServeCmd)
	rootCmd.AddCommand(cmds.LsServerCmd)
}

func main() {
	_ = rootCmd.Execute()

}
