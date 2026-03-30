package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jacksonfernando/qbridge/internal/config"
	dbpkg "github.com/jacksonfernando/qbridge/internal/db"
	"github.com/jacksonfernando/qbridge/internal/policy"
	"github.com/spf13/cobra"
)

var (
	queryProfile string
	queryDB      string
	queryJSON    bool
)

var queryCmd = &cobra.Command{
	Use:   "query --profile <profile> [--db <database>] \"<SQL>\"",
	Short: "Execute a SQL statement through a profile (policy enforced)",
	Long: `Execute SQL via a named profile. QBridge will:
  1. Verify the SQL operation is allowed by the profile
  2. Connect to the target database (or first DB in profile if --db is omitted)
  3. Return results as JSON

Example (read-only agent):
  qbridge query --profile readonly "SELECT id, name FROM users LIMIT 10"

Example (targeting a specific DB in the profile):
  qbridge query --profile analyst --db prod-postgres "SELECT count(*) FROM orders"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunQuery(mustLoadStore(), queryProfile, queryDB, args[0], os.Stdout)
	},
}

func init() {
	queryCmd.Flags().StringVarP(&queryProfile, "profile", "p", "", "Profile to use (required)")
	queryCmd.Flags().StringVarP(&queryDB, "db", "d", "", "Target database (defaults to first DB in profile)")
	queryCmd.Flags().BoolVar(&queryJSON, "json", true, "Output results as JSON (default: true)")
	_ = queryCmd.MarkFlagRequired("profile")
}

// runQuery contains the testable core of the query command.
func RunQuery(s config.Storer, profileName, targetDB, sql string, out io.Writer) error {
	profile, err := s.GetProfile(profileName)
	if err != nil {
		return err
	}

	if len(profile.Databases) == 0 {
		return fmt.Errorf("profile %q has no databases attached", profile.Name)
	}

	if targetDB == "" {
		targetDB = profile.Databases[0]
	}

	if err := policy.CheckDB(profile, targetDB); err != nil {
		return err
	}

	if err := policy.Check(profile, sql); err != nil {
		return err
	}

	dbCfg, err := s.GetDB(targetDB)
	if err != nil {
		return err
	}

	conn, err := dbpkg.Connect(dbCfg)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	result, err := dbpkg.Execute(conn, sql)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	output := map[string]interface{}{
		"profile":       profile.Name,
		"database":      targetDB,
		"columns":       result.Columns,
		"rows":          result.Rows,
		"rows_affected": result.Affected,
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
