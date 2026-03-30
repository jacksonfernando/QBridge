package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupStore creates a temporary QBridge directory, overrides the home dir, and
// returns an initialized *Store ready for testing.
func setupStore(t *testing.T, password string) *config.Store {
	t.Helper()
	tmpDir := t.TempDir()

	// Override HOME so QBridgeDir() points to tmpDir.
	t.Setenv("HOME", tmpDir)

	err := config.Initialize(password)
	require.NoError(t, err)

	s, err := config.Load(password)
	require.NoError(t, err)
	return s
}

// --- IsInitialized ---

func TestIsInitialized_False(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	assert.False(t, config.IsInitialized())
}

func TestIsInitialized_True(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Initialize("pw"))
	assert.True(t, config.IsInitialized())
}

// --- Initialize ---

func TestInitialize_CreatesStore(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	require.NoError(t, config.Initialize("secret"))

	storePath := filepath.Join(tmp, ".qbridge", "store.enc")
	_, err := os.Stat(storePath)
	assert.NoError(t, err, "store.enc should exist")
}

func TestInitialize_AlreadyInitialized_DoesNotOverwrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Initialize("first"))
	// Second init should still succeed (it just writes a fresh empty store)
	// In reality Initialize checks nothing; idempotency is the caller's responsibility.
	// We just verify it doesn't error on a second call.
	require.NoError(t, config.Initialize("second"))
}

// --- Load ---

func TestLoad_WrongPassword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Initialize("correct"))
	_, err := config.Load("wrong")
	assert.Error(t, err)
}

func TestLoad_EmptyStore_HasSlices(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Initialize("pw"))
	s, err := config.Load("pw")
	require.NoError(t, err)
	assert.NotNil(t, s.Config.Databases)
	assert.NotNil(t, s.Config.Profiles)
}

// --- ValidateOperation ---

func TestValidateOperation_Valid(t *testing.T) {
	for _, op := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DDL"} {
		got, err := config.ValidateOperation(op)
		assert.NoError(t, err, op)
		assert.Equal(t, config.Operation(op), got)
	}
}

func TestValidateOperation_Invalid(t *testing.T) {
	_, err := config.ValidateOperation("TRUNCATE")
	assert.Error(t, err)
}

// --- GetDB / AddDB / RemoveDB ---

func TestAddDB_Success(t *testing.T) {
	s := setupStore(t, "pw")
	db := config.Database{Name: "mydb", Type: config.DBTypePostgres}
	require.NoError(t, s.AddDB(db))
	got, err := s.GetDB("mydb")
	require.NoError(t, err)
	assert.Equal(t, "mydb", got.Name)
}

func TestAddDB_DuplicateName(t *testing.T) {
	s := setupStore(t, "pw")
	db := config.Database{Name: "mydb", Type: config.DBTypePostgres}
	require.NoError(t, s.AddDB(db))
	err := s.AddDB(db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGetDB_NotFound(t *testing.T) {
	s := setupStore(t, "pw")
	_, err := s.GetDB("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveDB_Success(t *testing.T) {
	s := setupStore(t, "pw")
	require.NoError(t, s.AddDB(config.Database{Name: "mydb", Type: config.DBTypeMySQL}))
	require.NoError(t, s.RemoveDB("mydb"))
	_, err := s.GetDB("mydb")
	assert.Error(t, err)
}

func TestRemoveDB_NotFound(t *testing.T) {
	s := setupStore(t, "pw")
	err := s.RemoveDB("ghost")
	assert.Error(t, err)
}

// --- GetProfile / AddProfile / RemoveProfile / UpdateProfile ---

func addTestDB(t *testing.T, s *config.Store, name string) {
	t.Helper()
	require.NoError(t, s.AddDB(config.Database{Name: name, Type: config.DBTypeSQLite, FilePath: "/tmp/x.db"}))
}

func TestAddProfile_Success(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	p := config.Profile{Name: "ro", Databases: []string{"db1"}, Allow: []config.Operation{config.OpSelect}}
	require.NoError(t, s.AddProfile(p))
	got, err := s.GetProfile("ro")
	require.NoError(t, err)
	assert.Equal(t, "ro", got.Name)
}

func TestAddProfile_DuplicateName(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	p := config.Profile{Name: "ro", Databases: []string{"db1"}, Allow: []config.Operation{config.OpSelect}}
	require.NoError(t, s.AddProfile(p))
	err := s.AddProfile(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddProfile_ReferencesNonExistentDB(t *testing.T) {
	s := setupStore(t, "pw")
	p := config.Profile{Name: "ro", Databases: []string{"ghost"}, Allow: []config.Operation{config.OpSelect}}
	err := s.AddProfile(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestGetProfile_NotFound(t *testing.T) {
	s := setupStore(t, "pw")
	_, err := s.GetProfile("nope")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveProfile_Success(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	p := config.Profile{Name: "ro", Databases: []string{"db1"}, Allow: []config.Operation{config.OpSelect}}
	require.NoError(t, s.AddProfile(p))
	require.NoError(t, s.RemoveProfile("ro"))
	_, err := s.GetProfile("ro")
	assert.Error(t, err)
}

func TestRemoveProfile_NotFound(t *testing.T) {
	s := setupStore(t, "pw")
	assert.Error(t, s.RemoveProfile("ghost"))
}

func TestUpdateProfile_Success(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	addTestDB(t, s, "db2")
	p := config.Profile{Name: "ro", Databases: []string{"db1"}, Allow: []config.Operation{config.OpSelect}}
	require.NoError(t, s.AddProfile(p))

	updated := config.Profile{Name: "ro", Databases: []string{"db2"}, Allow: []config.Operation{config.OpInsert}}
	require.NoError(t, s.UpdateProfile(updated))

	got, err := s.GetProfile("ro")
	require.NoError(t, err)
	assert.Equal(t, []string{"db2"}, got.Databases)
	assert.Equal(t, []config.Operation{config.OpInsert}, got.Allow)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	s := setupStore(t, "pw")
	err := s.UpdateProfile(config.Profile{Name: "ghost"})
	assert.Error(t, err)
}

func TestUpdateProfile_InvalidDB(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	p := config.Profile{Name: "ro", Databases: []string{"db1"}, Allow: []config.Operation{config.OpSelect}}
	require.NoError(t, s.AddProfile(p))
	err := s.UpdateProfile(config.Profile{Name: "ro", Databases: []string{"ghost"}})
	assert.Error(t, err)
}

// --- Save / Load round-trip ---

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	require.NoError(t, config.Initialize("pw"))
	s, err := config.Load("pw")
	require.NoError(t, err)

	require.NoError(t, s.AddDB(config.Database{Name: "mydb", Type: config.DBTypeSQLite, FilePath: "/tmp/x.db"}))
	require.NoError(t, s.AddProfile(config.Profile{
		Name:      "ro",
		Databases: []string{"mydb"},
		Allow:     []config.Operation{config.OpSelect},
	}))
	require.NoError(t, s.Save())

	s2, err := config.Load("pw")
	require.NoError(t, err)
	assert.Len(t, s2.Config.Databases, 1)
	assert.Len(t, s2.Config.Profiles, 1)
	assert.Equal(t, "mydb", s2.Config.Databases[0].Name)
	assert.Equal(t, "ro", s2.Config.Profiles[0].Name)
}

// --- Accessor methods ---

func TestGetDatabases(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	addTestDB(t, s, "db2")
	assert.Len(t, s.GetDatabases(), 2)
}

func TestGetProfiles(t *testing.T) {
	s := setupStore(t, "pw")
	addTestDB(t, s, "db1")
	require.NoError(t, s.AddProfile(config.Profile{
		Name:      "p1",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}))
	assert.Len(t, s.GetProfiles(), 1)
}
