package db_test

import (
	"os"
	"path/filepath"
	"testing"

	dbpkg "github.com/jacksonfernando/qbridge/internal/db"
	"github.com/jacksonfernando/qbridge/internal/config"
)

func TestSQLiteConnect(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{
		Name:     "test-sqlite",
		Type:     config.DBTypeSQLite,
		FilePath: dbPath,
	}

	conn, err := dbpkg.Connect(d)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	// Create table and insert.
	_, err = conn.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	_, err = conn.Exec("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}
}

func TestSQLiteExecuteSelect(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{
		Name:     "test-sqlite",
		Type:     config.DBTypeSQLite,
		FilePath: dbPath,
	}

	conn, err := dbpkg.Connect(d)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	conn.Exec("CREATE TABLE items (id INTEGER, label TEXT)")
	conn.Exec("INSERT INTO items VALUES (1, 'foo'), (2, 'bar')")

	result, err := dbpkg.Execute(conn, "SELECT id, label FROM items ORDER BY id")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(result.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(result.Columns))
	}
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result.Rows))
	}
}

func TestSQLiteExecuteInsert(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{
		Name:     "test-sqlite",
		Type:     config.DBTypeSQLite,
		FilePath: dbPath,
	}

	conn, err := dbpkg.Connect(d)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	conn.Exec("CREATE TABLE t (x INT)")
	result, err := dbpkg.Execute(conn, "INSERT INTO t VALUES (42)")
	if err != nil {
		t.Fatalf("Execute INSERT: %v", err)
	}
	if result.Affected != 1 {
		t.Errorf("expected 1 row affected, got %d", result.Affected)
	}
}

// Ensure unused import doesn't break.
var _ = os.Getenv
var _ = filepath.Join
