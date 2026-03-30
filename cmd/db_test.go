package cmd_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jacksonfernando/qbridge/cmd"
	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/jacksonfernando/qbridge/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- splitTrim (via parseOps) ---

func TestParseOps_Valid(t *testing.T) {
	// Exercise parseOps through the exported runProfileAdd path indirectly;
	// test it directly by calling runProfileAdd with controlled input.
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{{Name: "db1", Type: config.DBTypeSQLite}})
	m.On("AddProfile", config.Profile{
		Name:      "ro",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}).Return(nil)
	m.On("Save").Return(nil)

	in := strings.NewReader("db1\nSELECT\n")
	var out bytes.Buffer
	err := cmd.RunProfileAdd(m, "ro", in, &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), `"ro"`)
}

func TestParseOps_InvalidOp(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{{Name: "db1", Type: config.DBTypeSQLite}})

	in := strings.NewReader("db1\nSELECT,BADOP\n")
	var out bytes.Buffer
	err := cmd.RunProfileAdd(m, "ro", in, &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation")
}

// --- defaultPort ---

func TestDefaultPort(t *testing.T) {
	assert.Equal(t, "5432", cmd.DefaultPort(config.DBTypePostgres))
	assert.Equal(t, "3306", cmd.DefaultPort(config.DBTypeMySQL))
	assert.Equal(t, "", cmd.DefaultPort(config.DBTypeSQLite))
}

// --- formatOpsSlice ---

func TestFormatOpsSlice(t *testing.T) {
	got := cmd.FormatOpsSlice([]config.Operation{config.OpSelect, config.OpInsert})
	assert.Equal(t, "SELECT, INSERT", got)
}

func TestFormatOpsSlice_Empty(t *testing.T) {
	assert.Equal(t, "", cmd.FormatOpsSlice(nil))
}

// --- runDBList ---

func TestRunDBList_Empty(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{})
	var out bytes.Buffer
	require.NoError(t, cmd.RunDBList(m, &out))
	assert.Contains(t, out.String(), "No databases")
}

func TestRunDBList_WithDatabases(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{
		{Name: "prod", Type: config.DBTypePostgres, Host: "localhost", Port: 5432, User: "admin", DBName: "myapp"},
	})
	var out bytes.Buffer
	require.NoError(t, cmd.RunDBList(m, &out))
	assert.Contains(t, out.String(), "prod")
	assert.Contains(t, out.String(), "postgres")
}

func TestRunDBList_SQLite(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{
		{Name: "local", Type: config.DBTypeSQLite, FilePath: "/tmp/db.sqlite"},
	})
	var out bytes.Buffer
	require.NoError(t, cmd.RunDBList(m, &out))
	assert.Contains(t, out.String(), "/tmp/db.sqlite")
}

// --- runDBRemove ---

func TestRunDBRemove_Success(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveDB", "prod").Return(nil)
	m.On("GetProfiles").Return([]config.Profile{})
	m.On("Save").Return(nil)
	var out bytes.Buffer
	require.NoError(t, cmd.RunDBRemove(m, "prod", &out))
	assert.Contains(t, out.String(), `"prod"`)
}

func TestRunDBRemove_NotFound(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveDB", "ghost").Return(errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunDBRemove(m, "ghost", &out))
}

func TestRunDBRemove_SaveError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveDB", "prod").Return(nil)
	m.On("GetProfiles").Return([]config.Profile{})
	m.On("Save").Return(errors.New("disk full"))
	var out bytes.Buffer
	err := cmd.RunDBRemove(m, "prod", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save")
}

// --- runDBRemove cleans profile references ---

func TestRunDBRemove_CleansProfileReferences(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveDB", "db1").Return(nil)
	m.On("GetProfiles").Return([]config.Profile{
		{Name: "p1", Databases: []string{"db1", "db2"}, Allow: []config.Operation{config.OpSelect}},
	})
	// After filtering, db1 is removed leaving only db2
	m.On("UpdateProfile", config.Profile{
		Name:      "p1",
		Databases: []string{"db2"},
		Allow:     []config.Operation{config.OpSelect},
	}).Return(nil)
	m.On("Save").Return(nil)

	var out bytes.Buffer
	require.NoError(t, cmd.RunDBRemove(m, "db1", &out))
}

// --- runDBTest ---

type mockDBConn struct{ closed bool }

func (c *mockDBConn) Close() error { c.closed = true; return nil }

func TestRunDBTest_Success(t *testing.T) {
	m := mocks.NewMockStorer(t)
	db := &config.Database{Name: "prod", Type: config.DBTypePostgres}
	m.On("GetDB", "prod").Return(db, nil)

	conn := &mockDBConn{}
	connectFn := func(d *config.Database) (cmd.DBConn, error) {
		return conn, nil
	}

	var out bytes.Buffer
	require.NoError(t, cmd.RunDBTest(m, "prod", connectFn, &out))
	assert.Contains(t, out.String(), "successful")
	assert.True(t, conn.closed)
}

func TestRunDBTest_NotFound(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDB", "ghost").Return((*config.Database)(nil), errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunDBTest(m, "ghost", nil, &out))
}

func TestRunDBTest_ConnectFail(t *testing.T) {
	m := mocks.NewMockStorer(t)
	db := &config.Database{Name: "prod", Type: config.DBTypePostgres}
	m.On("GetDB", "prod").Return(db, nil)

	connectFn := func(d *config.Database) (cmd.DBConn, error) {
		return nil, errors.New("connection refused")
	}
	var out bytes.Buffer
	err := cmd.RunDBTest(m, "prod", connectFn, &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Connection failed")
}

// --- runDBAdd ---

func TestRunDBAdd_SQLite(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("AddDB", config.Database{
		Name:     "local",
		Type:     config.DBTypeSQLite,
		FilePath: "/tmp/local.db",
	}).Return(nil)
	m.On("Save").Return(nil)

	in := strings.NewReader("sqlite\n/tmp/local.db\n")
	var out bytes.Buffer
	require.NoError(t, cmd.RunDBAdd(m, "local", in, &out))
	assert.Contains(t, out.String(), `"local"`)
}

func TestRunDBAdd_DuplicateReturnsError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("AddDB", config.Database{
		Name:     "local",
		Type:     config.DBTypeSQLite,
		FilePath: "/tmp/x.db",
	}).Return(errors.New("already exists"))

	in := strings.NewReader("sqlite\n/tmp/x.db\n")
	var out bytes.Buffer
	assert.Error(t, cmd.RunDBAdd(m, "local", in, &out))
}
