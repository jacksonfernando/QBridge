package db_test

import (
	"path/filepath"
	"testing"

	"github.com/jacksonfernando/qbridge/internal/config"
	dbpkg "github.com/jacksonfernando/qbridge/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Unknown DB type ---

func TestConnect_UnknownType(t *testing.T) {
	d := &config.Database{Name: "x", Type: "oracle"}
	_, err := dbpkg.Connect(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// --- SQLite: empty file path ---

func TestConnect_SQLite_EmptyFilePath(t *testing.T) {
	d := &config.Database{Name: "x", Type: config.DBTypeSQLite, FilePath: ""}
	_, err := dbpkg.Connect(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filepath")
}

// --- Execute: WITH clause routes to queryRows ---

func TestExecute_WITH_Clause(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	result, err := dbpkg.Execute(conn, "WITH cte AS (SELECT 42 AS n) SELECT n FROM cte")
	require.NoError(t, err)
	assert.Equal(t, []string{"n"}, result.Columns)
	require.Len(t, result.Rows, 1)
}

// --- Execute: DDL statement ---

func TestExecute_DDL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	result, err := dbpkg.Execute(conn, "CREATE TABLE things (id INTEGER)")
	require.NoError(t, err)
	assert.Empty(t, result.Columns)
}

// --- Execute: multiple rows with different types ---

func TestExecute_MultipleRows(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	conn.Exec("CREATE TABLE nums (n INTEGER, s TEXT)")
	conn.Exec("INSERT INTO nums VALUES (1,'a'),(2,'b'),(3,'c')")

	result, err := dbpkg.Execute(conn, "SELECT n, s FROM nums ORDER BY n")
	require.NoError(t, err)
	assert.Len(t, result.Rows, 3)
	assert.Equal(t, []string{"n", "s"}, result.Columns)
}

// --- Execute: query error ---

func TestExecute_QueryError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	_, err = dbpkg.Execute(conn, "SELECT * FROM nonexistent_table")
	assert.Error(t, err)
}

// --- Execute: exec error ---

func TestExecute_ExecError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	_, err = dbpkg.Execute(conn, "INSERT INTO nonexistent VALUES (1)")
	assert.Error(t, err)
}

// --- Execute: rows_affected on multi-row delete ---

func TestExecute_RowsAffected(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d := &config.Database{Name: "t", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(d)
	require.NoError(t, err)
	defer conn.Close()

	conn.Exec("CREATE TABLE t (x INT)")
	conn.Exec("INSERT INTO t VALUES (1),(2),(3)")

	result, err := dbpkg.Execute(conn, "DELETE FROM t")
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Affected)
}
