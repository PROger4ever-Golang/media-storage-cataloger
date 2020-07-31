package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"media-storage-cataloger/commands"
	"media-storage-cataloger/config"
)

var rootCmd = &cobra.Command{
	Use:  "media-storage-cataloger",
	Long: `Rename for media-files and name them according to their taken dates.`,
}

func addCommands() {
	rootCmd.AddCommand(commands.GetRenameCommand())
}

func main() {
	addCommands()

	fmt.Println("Loading configs...")
	_, err := config.LoadConfig("config", "config/")
	if err != nil {
		log.Fatal(err)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
