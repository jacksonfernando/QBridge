package cmd

import (
	"fmt"
	"io"
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
		return RunDBAdd(mustLoadStore(), args[0], os.Stdin, os.Stdout)
	},
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunDBList(mustLoadStore(), os.Stdout)
	},
}

var dbRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a registered database credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunDBRemove(mustLoadStore(), args[0], os.Stdout)
	},
}

var dbTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test connectivity to a registered database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunDBTest(mustLoadStore(), args[0], dbConnect, os.Stdout)
	},
}

func init() {
	dbCmd.AddCommand(dbAddCmd)
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbRemoveCmd)
	dbCmd.AddCommand(dbTestCmd)
}

// runDBAdd contains the testable core of the db add command.
func RunDBAdd(s config.Storer, name string, in io.Reader, out io.Writer) error {
	dbEntry := config.Database{Name: name}

	dbEntry.Type = config.DBType(PromptChoose(out, in, "Database type", []string{"postgres", "mysql", "sqlite"}))

	if dbEntry.Type == config.DBTypeSQLite {
		dbEntry.FilePath = PromptValue(out, in, "File path", "")
	} else {
		dbEntry.Host = PromptValue(out, in, "Host", "")
		portStr := PromptValue(out, in, fmt.Sprintf("Port [%s]", DefaultPort(dbEntry.Type)), DefaultPort(dbEntry.Type))
		if portStr == "" {
			portStr = DefaultPort(dbEntry.Type)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
		dbEntry.Port = port
		dbEntry.User = PromptValue(out, in, "Username", "")
		fmt.Fprint(out, "Password: ")
		pw, err := ReadPasswordFrom(in)
		if err != nil {
			return err
		}
		fmt.Fprintln(out)
		dbEntry.Password = pw
		dbEntry.DBName = PromptValue(out, in, "Database name", "")

		if dbEntry.Type == config.DBTypePostgres {
			dbEntry.SSLMode = PromptValue(out, in, "SSL mode (disable/require/prefer/verify-full) [prefer]", "prefer")
			if dbEntry.SSLMode == "" {
				dbEntry.SSLMode = "prefer"
			}
		}
	}

	if err := s.AddDB(dbEntry); err != nil {
		return err
	}
	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Fprintf(out, "✓ Database %q registered.\n", name)
	return nil
}

// runDBList contains the testable core of the db list command.
func RunDBList(s config.Storer, out io.Writer) error {
	dbs := s.GetDatabases()
	if len(dbs) == 0 {
		fmt.Fprintln(out, "No databases registered. Run: qbridge db add <name>")
		return nil
	}
	table := tablewriter.NewWriter(out)
	table.Header([]string{"Name", "Type", "Host / Path", "Port", "User", "Database"})
	for _, d := range dbs {
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
}

// ConnectFn is a function type matching db.Connect so it can be swapped in tests.
type ConnectFn func(d *config.Database) (DBConn, error)

// DBConn is the minimal interface of *sql.DB used by runDBTest.
type DBConn interface {
	Close() error
}

// runDBRemove contains the testable core of the db remove command.
func RunDBRemove(s config.Storer, name string, out io.Writer) error {
	if err := s.RemoveDB(name); err != nil {
		return err
	}

	// Clean up profile references.
	for _, p := range s.GetProfiles() {
		filtered := p.Databases[:0]
		for _, d := range p.Databases {
			if d != name {
				filtered = append(filtered, d)
			}
		}
		p.Databases = filtered
		_ = s.UpdateProfile(p)
	}

	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Fprintf(out, "✓ Database %q removed.\n", name)
	return nil
}

// runDBTest contains the testable core of the db test command.
func RunDBTest(s config.Storer, name string, connect ConnectFn, out io.Writer) error {
	d, err := s.GetDB(name)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Connecting to %q (%s)...\n", d.Name, d.Type)
	conn, err := connect(d)
	if err != nil {
		return fmt.Errorf("✗ Connection failed: %w", err)
	}
	conn.Close()
	fmt.Fprintln(out, "✓ Connection successful.")
	return nil
}

// --- helpers ---

func DefaultPort(t config.DBType) string {
	switch t {
	case config.DBTypePostgres:
		return "5432"
	case config.DBTypeMySQL:
		return "3306"
	}
	return ""
}

// promptValue prints a label to out, reads one token from in, and falls back to def if empty.
func PromptValue(out io.Writer, in io.Reader, label, def string) string {
	fmt.Fprintf(out, "%s: ", label)
	var val string
	fmt.Fscan(in, &val)
	val = strings.TrimSpace(val)
	if val == "" {
		return def
	}
	return val
}

// promptChoose loops until the user enters one of options.
func PromptChoose(out io.Writer, in io.Reader, label string, options []string) string {
	for {
		fmt.Fprintf(out, "%s (%s): ", label, strings.Join(options, "/"))
		var val string
		fmt.Fscan(in, &val)
		val = strings.ToLower(strings.TrimSpace(val))
		for _, o := range options {
			if val == o {
				return val
			}
		}
		fmt.Fprintf(out, "  Please choose one of: %s\n", strings.Join(options, ", "))
	}
}

// readPasswordFrom reads a password from r without echo (falls back to plain read for non-TTY).
func ReadPasswordFrom(r io.Reader) (string, error) {
	if r == os.Stdin {
		return readPassword()
	}
	var pw string
	_, err := fmt.Fscan(r, &pw)
	return strings.TrimSpace(pw), err
}

// dbConnect wraps db.Connect to satisfy ConnectFn signature.
func dbConnect(d *config.Database) (DBConn, error) {
	return db.Connect(d)
}

