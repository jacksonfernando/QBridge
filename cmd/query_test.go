package cmd_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jacksonfernando/qbridge/cmd"
	"github.com/jacksonfernando/qbridge/internal/config"
	dbpkg "github.com/jacksonfernando/qbridge/internal/db"
	"github.com/jacksonfernando/qbridge/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReadonlyProfile() config.Profile {
	return config.Profile{
		Name:      "ro",
		Databases: []string{"testdb"},
		Allow:     []config.Operation{config.OpSelect},
	}
}

// --- profile not found ---

func TestRunQuery_ProfileNotFound(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetProfile", "ro").Return((*config.Profile)(nil), errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunQuery(m, "ro", "", "SELECT 1", &out))
}

// --- profile has no databases ---

func TestRunQuery_ProfileNoDatabases(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := config.Profile{Name: "empty", Databases: []string{}, Allow: []config.Operation{config.OpSelect}}
	m.On("GetProfile", "empty").Return(&p, nil)
	var out bytes.Buffer
	err := cmd.RunQuery(m, "empty", "", "SELECT 1", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no databases")
}

// --- database not in profile ---

func TestRunQuery_DBNotInProfile(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	var out bytes.Buffer
	err := cmd.RunQuery(m, "ro", "otherdb", "SELECT 1", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not accessible")
}

// --- operation denied ---

func TestRunQuery_OperationDenied(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	var out bytes.Buffer
	err := cmd.RunQuery(m, "ro", "testdb", "DELETE FROM users", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

// --- defaults to first DB when --db is empty ---

func TestRunQuery_DefaultsToFirstDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a real SQLite DB.
	realDB := &config.Database{Name: "testdb", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(realDB)
	require.NoError(t, err)
	conn.Exec("CREATE TABLE t (x INT)")
	conn.Close()

	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	m.On("GetDB", "testdb").Return(realDB, nil) // called because targetDB defaults to profile.Databases[0]

	var out bytes.Buffer
	require.NoError(t, cmd.RunQuery(m, "ro", "", "SELECT * FROM t", &out))
	assert.Contains(t, out.String(), `"profile"`)
	assert.Contains(t, out.String(), `"testdb"`)
}

// --- successful query returns JSON ---

func TestRunQuery_Success_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	realDB := &config.Database{Name: "testdb", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(realDB)
	require.NoError(t, err)
	conn.Exec("CREATE TABLE users (id INTEGER, name TEXT)")
	conn.Exec("INSERT INTO users VALUES (1, 'Alice')")
	conn.Close()

	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	m.On("GetDB", "testdb").Return(realDB, nil)

	var out bytes.Buffer
	require.NoError(t, cmd.RunQuery(m, "ro", "testdb", "SELECT id, name FROM users", &out))

	body := out.String()
	assert.Contains(t, body, `"columns"`)
	assert.Contains(t, body, `"rows"`)
	assert.Contains(t, body, "Alice")
}

// --- unknown SQL statement ---

func TestRunQuery_UnknownSQL(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := config.Profile{Name: "all", Databases: []string{"testdb"}, Allow: config.AllOperations}
	m.On("GetProfile", "all").Return(&p, nil)
	var out bytes.Buffer
	err := cmd.RunQuery(m, "all", "testdb", "EXPLAIN SELECT 1", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy check failed")
}

// --- GetDB error ---

func TestRunQuery_GetDBError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	m.On("GetDB", "testdb").Return((*config.Database)(nil), errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunQuery(m, "ro", "testdb", "SELECT 1", &out))
}

// --- runQuery with INSERT on write profile ---

func TestRunQuery_InsertAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	realDB := &config.Database{Name: "testdb", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(realDB)
	require.NoError(t, err)
	conn.Exec("CREATE TABLE t (x INT)")
	conn.Close()

	m := mocks.NewMockStorer(t)
	p := config.Profile{
		Name:      "rw",
		Databases: []string{"testdb"},
		Allow:     []config.Operation{config.OpSelect, config.OpInsert},
	}
	m.On("GetProfile", "rw").Return(&p, nil)
	m.On("GetDB", "testdb").Return(realDB, nil)

	var out bytes.Buffer
	require.NoError(t, cmd.RunQuery(m, "rw", "testdb", "INSERT INTO t VALUES (42)", &out))

	// Verify the output has rows_affected.
	assert.Contains(t, out.String(), `"rows_affected"`)
}

// --- WITH clause counts as SELECT ---

func TestRunQuery_WithClause_TreatedAsSelect(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	realDB := &config.Database{Name: "testdb", Type: config.DBTypeSQLite, FilePath: dbPath}
	conn, err := dbpkg.Connect(realDB)
	require.NoError(t, err)
	conn.Close()

	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	m.On("GetDB", "testdb").Return(realDB, nil)

	var out bytes.Buffer
	sql := "WITH cte AS (SELECT 1 AS n) SELECT n FROM cte"
	// SQLite supports CTEs; this should succeed with SELECT-only profile.
	require.NoError(t, cmd.RunQuery(m, "ro", "testdb", sql, &out))
	assert.Contains(t, out.String(), `"columns"`)
}

// --- profileEdit blank input handling ---
// Exercises that blank prompts use Fscan which skips whitespace — confirm no crash.
func TestRunQuery_DDLDeniedOnReadonly(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := newReadonlyProfile()
	m.On("GetProfile", "ro").Return(&p, nil)
	var out bytes.Buffer
	err := cmd.RunQuery(m, "ro", "testdb", "CREATE TABLE x (id INT)", &out)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "not allowed")
}
