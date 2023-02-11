package main

import (
	"github.com/spf13/cobra"
	"github.com/wesen/parka/cmd/parka/cmds"
)

var rootCmd = &cobra.Command{
	Use:   "parka",
	Short: "parka keeps command-line applications warm and fuzzy by giving them a fluffy web API",
}

func init() {
	rootCmd.AddCommand(cmds.ServeCmd)
	rootCmd.AddCommand(cmds.LsServerCmd)
}

func main() {
	_ = rootCmd.Execute()

}
