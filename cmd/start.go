package cmd

import (
	"payment-service/internal/app"

	"github.com/spf13/cobra"
)

// go run main.go start
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start",
	Long:  `start`,
	Run: func(cmd *cobra.Command, args []string) {
		// Call Func Route API
		app.RunServer()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
