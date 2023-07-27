package cmd

import (
	"github.com/spf13/cobra"
	"gocrawler/cmd/master"
	"gocrawler/cmd/worker"
	"gocrawler/version"
)

var masterCmd = &cobra.Command{
	Use:   "master",
	Short: "run master service.",
	Long:  "run master service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		master.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version.",
	Long:  "print version.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		version.Printer()
	},
}

func Execute() {
	var rootCmd = &cobra.Command{Use: "crawler"}
	rootCmd.AddCommand(masterCmd, worker.WorkerCmd, versionCmd)
	rootCmd.Execute()
}
