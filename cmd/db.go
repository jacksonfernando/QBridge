package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/jacksonfernando/qbridge/internal/db"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage database credentials",
}

var dbAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Register a new database credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		name := args[0]

		dbEntry := config.Database{Name: name}

		// Choose type.
		dbEntry.Type = config.DBType(mustChoose("Database type", []string{"postgres", "mysql", "sqlite"}))

		if dbEntry.Type == config.DBTypeSQLite {
			dbEntry.FilePath = mustPrompt("File path")
		} else {
			dbEntry.Host = mustPrompt("Host")
			portStr := mustPromptDefault("Port", defaultPort(dbEntry.Type))
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
			dbEntry.Port = port
			dbEntry.User = mustPrompt("Username")
			fmt.Print("Password: ")
			pw, err := readPassword()
			if err != nil {
				return err
			}
			fmt.Println()
			dbEntry.Password = pw
			dbEntry.DBName = mustPrompt("Database name")

			if dbEntry.Type == config.DBTypePostgres {
				dbEntry.SSLMode = mustPromptDefault("SSL mode (disable/require/prefer/verify-full)", "prefer")
			}
		}

		if err := s.AddDB(dbEntry); err != nil {
			return err
		}
		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("✓ Database %q registered.\n", name)
		return nil
	},
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		if len(s.Config.Databases) == 0 {
			fmt.Println("No databases registered. Run: qbridge db add <name>")
			return nil
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"Name", "Type", "Host / Path", "Port", "User", "Database"})
		for _, d := range s.Config.Databases {
			host := d.Host
			port := ""
			if d.Type == config.DBTypeSQLite {
				host = d.FilePath
			} else {
				port = strconv.Itoa(d.Port)
			}
			table.Append([]string{d.Name, string(d.Type), host, port, d.User, d.DBName})
		}
		table.Render()
		return nil
	},
}

var dbRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a registered database credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		if err := s.RemoveDB(args[0]); err != nil {
			return err
		}

		// Also clean up any profile references.
		for i := range s.Config.Profiles {
			p := &s.Config.Profiles[i]
			filtered := p.Databases[:0]
			for _, d := range p.Databases {
				if d != args[0] {
					filtered = append(filtered, d)
				}
			}
			p.Databases = filtered
		}

		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("✓ Database %q removed.\n", args[0])
		return nil
	},
}

var dbTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test connectivity to a registered database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		d, err := s.GetDB(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Connecting to %q (%s)...\n", d.Name, d.Type)
		conn, err := db.Connect(d)
		if err != nil {
			return fmt.Errorf("✗ Connection failed: %w", err)
		}
		conn.Close()
		fmt.Println("✓ Connection successful.")
		return nil
	},
}

func init() {
	dbCmd.AddCommand(dbAddCmd)
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbRemoveCmd)
	dbCmd.AddCommand(dbTestCmd)
}

// --- helpers ---

func defaultPort(t config.DBType) string {
	switch t {
	case config.DBTypePostgres:
		return "5432"
	case config.DBTypeMySQL:
		return "3306"
	}
	return ""
}

func mustPrompt(label string) string {
	fmt.Printf("%s: ", label)
	var val string
	fmt.Fscan(os.Stdin, &val)
	return strings.TrimSpace(val)
}

func mustPromptDefault(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	var val string
	fmt.Fscan(os.Stdin, &val)
	val = strings.TrimSpace(val)
	if val == "" {
		return def
	}
	return val
}

func mustChoose(label string, options []string) string {
	for {
		fmt.Printf("%s (%s): ", label, strings.Join(options, "/"))
		var val string
		fmt.Fscan(os.Stdin, &val)
		val = strings.ToLower(strings.TrimSpace(val))
		for _, o := range options {
			if val == o {
				return val
			}
		}
		fmt.Printf("  Please choose one of: %s\n", strings.Join(options, ", "))
	}
}
