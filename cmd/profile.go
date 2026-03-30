package cmd

import (
	"fmt"
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
		s := mustLoadStore()
		name := args[0]

		if len(s.Config.Databases) == 0 {
			return fmt.Errorf("no databases registered yet. Run: qbridge db add <name>")
		}

		fmt.Println("Available databases:", listDBNames(s))
		fmt.Print("Databases to include (comma-separated): ")
		var dbsInput string
		fmt.Fscan(os.Stdin, &dbsInput)
		dbs := splitTrim(dbsInput)

		fmt.Printf("Allowed operations — choices: %s\n", formatAllOps())
		fmt.Print("Allow (comma-separated): ")
		var opsInput string
		fmt.Fscan(os.Stdin, &opsInput)
		ops, err := parseOps(opsInput)
		if err != nil {
			return err
		}

		p := config.Profile{Name: name, Databases: dbs, Allow: ops}
		if err := s.AddProfile(p); err != nil {
			return err
		}
		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("✓ Profile %q created.\n", name)
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		if len(s.Config.Profiles) == 0 {
			fmt.Println("No profiles defined. Run: qbridge profile add <name>")
			return nil
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"Name", "Databases", "Allowed Operations"})
		for _, p := range s.Config.Profiles {
			table.Append([]string{
				p.Name,
				strings.Join(p.Databases, ", "),
				formatOpsSlice(p.Allow),
			})
		}
		table.Render()
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		p, err := s.GetProfile(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Profile:   %s\n", p.Name)
		fmt.Printf("Databases: %s\n", strings.Join(p.Databases, ", "))
		fmt.Printf("Allow:     %s\n", formatOpsSlice(p.Allow))
		return nil
	},
}

var profileEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing profile's databases and permissions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		p, err := s.GetProfile(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Current databases: %s\n", strings.Join(p.Databases, ", "))
		fmt.Println("Available databases:", listDBNames(s))
		fmt.Print("New databases (comma-separated, leave blank to keep current): ")
		var dbsInput string
		fmt.Fscan(os.Stdin, &dbsInput)
		if strings.TrimSpace(dbsInput) != "" {
			p.Databases = splitTrim(dbsInput)
		}

		fmt.Printf("Current allow: %s\n", formatOpsSlice(p.Allow))
		fmt.Printf("Allowed operations — choices: %s\n", formatAllOps())
		fmt.Print("New allow (comma-separated, leave blank to keep current): ")
		var opsInput string
		fmt.Fscan(os.Stdin, &opsInput)
		if strings.TrimSpace(opsInput) != "" {
			ops, err := parseOps(opsInput)
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
		fmt.Printf("✓ Profile %q updated.\n", p.Name)
		return nil
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustLoadStore()
		if err := s.RemoveProfile(args[0]); err != nil {
			return err
		}
		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("✓ Profile %q removed.\n", args[0])
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileEditCmd)
	profileCmd.AddCommand(profileRemoveCmd)
}

// --- helpers ---

func listDBNames(s *config.Store) string {
	names := make([]string, len(s.Config.Databases))
	for i, d := range s.Config.Databases {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

func formatAllOps() string {
	parts := make([]string, len(config.AllOperations))
	for i, o := range config.AllOperations {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}

func formatOpsSlice(ops []config.Operation) string {
	parts := make([]string, len(ops))
	for i, o := range ops {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}

func parseOps(input string) ([]config.Operation, error) {
	parts := splitTrim(input)
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

func splitTrim(s string) []string {
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
