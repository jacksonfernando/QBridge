package policy

import (
	"fmt"
	"strings"

	"github.com/jacksonfernando/qbridge/internal/config"
)

// statementPrefixes maps SQL keywords to their granular Operation.
var statementPrefixes = map[string]config.Operation{
	"SELECT":   config.OpSelect,
	"WITH":     config.OpSelect, // CTEs that read
	"INSERT":   config.OpInsert,
	"UPDATE":   config.OpUpdate,
	"DELETE":   config.OpDelete,
	"CREATE":   config.OpCreate,
	"DROP":     config.OpDrop,
	"ALTER":    config.OpAlter,
	"RENAME":   config.OpRename,
	"TRUNCATE": config.OpTruncate,
}

// ClassifySQL returns the Operation class for a given SQL statement.
func ClassifySQL(sql string) (config.Operation, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return "", fmt.Errorf("empty SQL statement")
	}

	// Extract the first keyword (handle comments by skipping -- and /* */ blocks naively).
	keyword := strings.ToUpper(firstKeyword(trimmed))

	op, ok := statementPrefixes[keyword]
	if !ok {
		return "", fmt.Errorf("unsupported or unrecognised SQL statement type: %q", keyword)
	}
	return op, nil
}

// firstKeyword extracts the first SQL token, skipping leading block/line comments.
func firstKeyword(sql string) string {
	s := sql
	for {
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "--") {
			// Skip to end of line.
			idx := strings.Index(s, "\n")
			if idx == -1 {
				return ""
			}
			s = s[idx+1:]
			continue
		}
		if strings.HasPrefix(s, "/*") {
			idx := strings.Index(s, "*/")
			if idx == -1 {
				return ""
			}
			s = s[idx+2:]
			continue
		}
		break
	}
	// Take the first word.
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Check returns nil if the profile allows the given SQL, or a descriptive error.
func Check(profile *config.Profile, sql string) error {
	op, err := ClassifySQL(sql)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}

	for _, allowed := range profile.Allow {
		if allowed == op {
			return nil
		}
	}

	return fmt.Errorf(
		"operation %q is not allowed by profile %q (allowed: %s)",
		op, profile.Name, formatOps(profile.Allow),
	)
}

// CheckDB returns nil if the profile has access to the named database.
func CheckDB(profile *config.Profile, dbName string) error {
	for _, d := range profile.Databases {
		if d == dbName {
			return nil
		}
	}
	return fmt.Errorf(
		"database %q is not accessible by profile %q (accessible: %s)",
		dbName, profile.Name, strings.Join(profile.Databases, ", "),
	)
}

func formatOps(ops []config.Operation) string {
	parts := make([]string, len(ops))
	for i, o := range ops {
		parts[i] = string(o)
	}
	return strings.Join(parts, ", ")
}
