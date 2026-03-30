package policy_test

import (
	"testing"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/jacksonfernando/qbridge/internal/policy"
)

func TestClassifySQL(t *testing.T) {
	cases := []struct {
		sql      string
		expected config.Operation
	}{
		{"SELECT * FROM users", config.OpSelect},
		{"  select id from t", config.OpSelect},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", config.OpSelect},
		{"INSERT INTO t VALUES (1)", config.OpInsert},
		{"UPDATE t SET x=1", config.OpUpdate},
		{"DELETE FROM t", config.OpDelete},
		{"CREATE TABLE t (id INT)", config.OpCreate},
		{"DROP TABLE t", config.OpDrop},
		{"ALTER TABLE t ADD COLUMN x INT", config.OpAlter},
		{"TRUNCATE TABLE t", config.OpTruncate},
		// Comments before keyword
		{"-- comment\nSELECT 1", config.OpSelect},
		{"/* block */ INSERT INTO t VALUES (1)", config.OpInsert},
	}

	for _, tc := range cases {
		op, err := policy.ClassifySQL(tc.sql)
		if err != nil {
			t.Errorf("ClassifySQL(%q) error: %v", tc.sql, err)
			continue
		}
		if op != tc.expected {
			t.Errorf("ClassifySQL(%q) = %q, want %q", tc.sql, op, tc.expected)
		}
	}
}

func TestClassifySQL_Empty(t *testing.T) {
	_, err := policy.ClassifySQL("")
	if err == nil {
		t.Error("expected error for empty SQL, got nil")
	}
}

func TestClassifySQL_Unknown(t *testing.T) {
	_, err := policy.ClassifySQL("EXPLAIN SELECT 1")
	if err == nil {
		t.Error("expected error for unknown statement, got nil")
	}
}

func makeProfile(name string, dbs []string, ops ...config.Operation) *config.Profile {
	return &config.Profile{Name: name, Databases: dbs, Allow: ops}
}

func TestCheck_Allowed(t *testing.T) {
	p := makeProfile("readonly", []string{"mydb"}, config.OpSelect)
	if err := policy.Check(p, "SELECT 1"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheck_Denied(t *testing.T) {
	p := makeProfile("readonly", []string{"mydb"}, config.OpSelect)
	if err := policy.Check(p, "INSERT INTO t VALUES (1)"); err == nil {
		t.Error("expected denial for INSERT on read-only profile")
	}
}

func TestCheck_MultipleOps(t *testing.T) {
	p := makeProfile("readwrite", []string{"mydb"}, config.OpSelect, config.OpInsert, config.OpUpdate)
	for _, sql := range []string{"SELECT 1", "INSERT INTO t VALUES (1)", "UPDATE t SET x=1"} {
		if err := policy.Check(p, sql); err != nil {
			t.Errorf("expected %q allowed, got: %v", sql, err)
		}
	}
	if err := policy.Check(p, "DELETE FROM t"); err == nil {
		t.Error("expected DELETE to be denied")
	}
}

func TestCheckDB_Allowed(t *testing.T) {
	p := makeProfile("p", []string{"db1", "db2"}, config.OpSelect)
	if err := policy.CheckDB(p, "db1"); err != nil {
		t.Errorf("expected db1 to be allowed: %v", err)
	}
}

func TestCheckDB_Denied(t *testing.T) {
	p := makeProfile("p", []string{"db1"}, config.OpSelect)
	if err := policy.CheckDB(p, "db2"); err == nil {
		t.Error("expected db2 to be denied")
	}
}
