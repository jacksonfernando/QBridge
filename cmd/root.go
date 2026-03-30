package cmd

import (
	"fmt"
	"os"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/spf13/cobra"
)

var store *config.Store

var rootCmd = &cobra.Command{
	Use:   "qbridge",
	Short: "QBridge — a control layer between AI agents and databases",
	Long: `QBridge enforces database access policies for AI agents.

Register database credentials, define profiles with allowed operations,
then let AI agents query through profiles — safely.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(queryCmd)
}

// mustLoadStore loads the encrypted store with the master password.
// Exits with a helpful message if not initialized or password is wrong.
func mustLoadStore() *config.Store {
	if !config.IsInitialized() {
		fmt.Fprintln(os.Stderr, "QBridge is not initialized. Run: qbridge init")
		os.Exit(1)
	}

	password := os.Getenv("QBRIDGE_PASSWORD")
	if password == "" {
		fmt.Fprint(os.Stderr, "Master password: ")
		pw, err := readPassword()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading password:", err)
			os.Exit(1)
		}
		password = pw
		fmt.Fprintln(os.Stderr) // newline after hidden input
	}

	s, err := config.Load(password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return s
}
