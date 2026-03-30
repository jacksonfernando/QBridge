package cmd

import (
	"fmt"
	"os"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize QBridge and set a master password",
	Long: `Creates ~/.qbridge/ and initializes an encrypted credential store.
The master password is used to encrypt all database credentials.

You can set QBRIDGE_PASSWORD environment variable to avoid interactive prompts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.IsInitialized() {
			return fmt.Errorf("QBridge is already initialized. To reset, delete ~/.qbridge/")
		}

		fmt.Print("Choose a master password: ")
		pw1, err := readPassword()
		if err != nil {
			return err
		}
		fmt.Println()

		fmt.Print("Confirm master password: ")
		pw2, err := readPassword()
		if err != nil {
			return err
		}
		fmt.Println()

		if pw1 != pw2 {
			return fmt.Errorf("passwords do not match")
		}
		if pw1 == "" {
			return fmt.Errorf("password cannot be empty")
		}

		if err := config.Initialize(pw1); err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		fmt.Println("✓ QBridge initialized. Run 'qbridge db add' to register your first database.")
		return nil
	},
}

// readPassword reads a password without echoing it to the terminal.
// Falls back to plain readline when not on a real TTY (e.g., piped input in tests).
func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		return string(pw), nil
	}
	// Not a TTY — read a line directly (useful for scripting/tests).
	var pw string
	_, err := fmt.Fscan(os.Stdin, &pw)
	return pw, err
}
