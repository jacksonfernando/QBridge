package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage access profiles",
}

var profileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new access profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunProfileAdd(mustLoadStore(), args[0], os.Stdin, os.Stdout)
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunProfileList(mustLoadStore(), os.Stdout)
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunProfileShow(mustLoadStore(), args[0], os.Stdout)
	},
}

var profileEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing profile's databases and permissions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunProfileEdit(mustLoadStore(), args[0], os.Stdin, os.Stdout)
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunProfileRemove(mustLoadStore(), args[0], os.Stdout)
	},
}

func init() {
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileRemoveCmd)
}

// runProfileAdd contains the testable core of profile add.
func RunProfileAdd(s config.Storer, name string, in io.Reader, out io.Writer) error {
	dbs := s.GetDatabases()
	if len(dbs) == 0 {
		return fmt.Errorf("no databases registered yet. Run: qbridge db add <name>")
	}

	fmt.Fprintln(out, "Available databases:", ListDBNames(s))
	fmt.Fprint(out, "Databases to include (comma-separated): ")
	var dbsInput string
	fmt.Fscan(in, &dbsInput)
	selectedDBs := SplitTrim(dbsInput)

	fmt.Fprintf(out, "Allowed operations — choices: %s\n", FormatAllOps())
	fmt.Fprint(out, "Allow (comma-separated): ")
	var opsInput string
	fmt.Fscan(in, &opsInput)
	ops, err := ParseOps(opsInput)
	if err != nil {
		return err
	}

	p := config.Profile{Name: name, Databases: selectedDBs, Allow: ops}
	if err := s.AddProfile(p); err != nil {
		return err
	}
	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Fprintf(out, "✓ Profile %q created.\n", name)
	return nil
}

// runProfileList contains the testable core of profile list.
func RunProfileList(s config.Storer, out io.Writer) error {
	profiles := s.GetProfiles()
	if len(profiles) == 0 {
		fmt.Fprintln(out, "No profiles defined. Run: qbridge profile add <name>")
		return nil
	}
	table := tablewriter.NewWriter(out)
	table.Header([]string{"Name", "Databases", "Allowed Operations"})
	for _, p := range profiles {
		table.Append([]string{
			p.Name,
			strings.Join(p.Databases, ", "),
			FormatOpsSlice(p.Allow),
		})
	}
	table.Render()
	return nil
}

// runProfileShow contains the testable core of profile show.
func RunProfileShow(s config.Storer, name string, out io.Writer) error {
	p, err := s.GetProfile(name)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Profile:   %s\n", p.Name)
	fmt.Fprintf(out, "Databases: %s\n", strings.Join(p.Databases, ", "))
	fmt.Fprintf(out, "Allow:     %s\n", FormatOpsSlice(p.Allow))
	return nil
}

// runProfileEdit contains the testable core of profile edit.
func RunProfileEdit(s config.Storer, name string, in io.Reader, out io.Writer) error {
	p, err := s.GetProfile(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Current databases: %s\n", strings.Join(p.Databases, ", "))
	fmt.Fprintln(out, "Available databases:", ListDBNames(s))
	fmt.Fprint(out, "New databases (comma-separated, leave blank to keep current): ")
	var dbsInput string
	fmt.Fscan(in, &dbsInput)
	if strings.TrimSpace(dbsInput) != "" {
		p.Databases = SplitTrim(dbsInput)
	}

	fmt.Fprintf(out, "Current allow: %s\n", FormatOpsSlice(p.Allow))
	fmt.Fprintf(out, "Allowed operations — choices: %s\n", FormatAllOps())
	fmt.Fprint(out, "New allow (comma-separated, leave blank to keep current): ")
	var opsInput string
	fmt.Fscan(in, &opsInput)
	if strings.TrimSpace(opsInput) != "" {
		ops, err := ParseOps(opsInput)
		if err != nil {
			return err
		}
		p.Allow = ops
	}

	if err := s.UpdateProfile(*p); err != nil {
		return err
	}
	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Fprintf(out, "✓ Profile %q updated.\n", p.Name)
	return nil
}

// runProfileRemove contains the testable core of profile remove.
func RunProfileRemove(s config.Storer, name string, out io.Writer) error {
	if err := s.RemoveProfile(name); err != nil {
		return err
	}
	if err := s.Save(); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	fmt.Fprintf(out, "✓ Profile %q removed.\n", name)
	return nil
}

// --- helpers ---

func ListDBNames(s config.Storer) string {
	names := make([]string, len(s.GetDatabases()))
	for i, d := range s.GetDatabases() {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

func FormatAllOps() string {
	parts := make([]string, len(config.AllOperations))
	for i, o := range config.AllOperations {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}

func FormatOpsSlice(ops []config.Operation) string {
	parts := make([]string, len(ops))
	for i, o := range ops {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}

func ParseOps(input string) ([]config.Operation, error) {
	parts := SplitTrim(input)
	if len(parts) == 0 {
		return nil, fmt.Errorf("at least one operation must be specified")
	}
	var ops []config.Operation
	for _, p := range parts {
		op, err := config.ValidateOperation(strings.ToUpper(p))
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func SplitTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}
