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

// --- runProfileList ---

func TestRunProfileList_Empty(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetProfiles").Return([]config.Profile{})
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileList(m, &out))
	assert.Contains(t, out.String(), "No profiles")
}

func TestRunProfileList_WithProfiles(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetProfiles").Return([]config.Profile{
		{Name: "readonly", Databases: []string{"prod"}, Allow: []config.Operation{config.OpSelect}},
	})
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileList(m, &out))
	assert.Contains(t, out.String(), "readonly")
	assert.Contains(t, out.String(), "SELECT")
}

// --- runProfileShow ---

func TestRunProfileShow_Success(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetProfile", "readonly").Return(&config.Profile{
		Name:      "readonly",
		Databases: []string{"prod"},
		Allow:     []config.Operation{config.OpSelect},
	}, nil)
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileShow(m, "readonly", &out))
	assert.Contains(t, out.String(), "readonly")
	assert.Contains(t, out.String(), "prod")
	assert.Contains(t, out.String(), "SELECT")
}

func TestRunProfileShow_NotFound(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetProfile", "nope").Return((*config.Profile)(nil), errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunProfileShow(m, "nope", &out))
}

// --- runProfileAdd ---

func TestRunProfileAdd_Success(t *testing.T) {
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
	require.NoError(t, cmd.RunProfileAdd(m, "ro", in, &out))
	assert.Contains(t, out.String(), `"ro"`)
}

func TestRunProfileAdd_NoDatabases(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{})
	var out bytes.Buffer
	err := cmd.RunProfileAdd(m, "ro", strings.NewReader(""), &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no databases")
}

func TestRunProfileAdd_AddProfileError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{{Name: "db1", Type: config.DBTypeSQLite}})
	m.On("AddProfile", config.Profile{
		Name:      "ro",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}).Return(errors.New("already exists"))

	in := strings.NewReader("db1\nSELECT\n")
	var out bytes.Buffer
	assert.Error(t, cmd.RunProfileAdd(m, "ro", in, &out))
}

func TestRunProfileAdd_SaveError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("GetDatabases").Return([]config.Database{{Name: "db1", Type: config.DBTypeSQLite}})
	m.On("AddProfile", config.Profile{
		Name:      "ro",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}).Return(nil)
	m.On("Save").Return(errors.New("disk full"))

	in := strings.NewReader("db1\nSELECT\n")
	var out bytes.Buffer
	err := cmd.RunProfileAdd(m, "ro", in, &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save")
}

// --- runProfileRemove ---

func TestRunProfileRemove_Success(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveProfile", "ro").Return(nil)
	m.On("Save").Return(nil)
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileRemove(m, "ro", &out))
	assert.Contains(t, out.String(), `"ro"`)
}

func TestRunProfileRemove_NotFound(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveProfile", "ghost").Return(errors.New("not found"))
	var out bytes.Buffer
	assert.Error(t, cmd.RunProfileRemove(m, "ghost", &out))
}

func TestRunProfileRemove_SaveError(t *testing.T) {
	m := mocks.NewMockStorer(t)
	m.On("RemoveProfile", "ro").Return(nil)
	m.On("Save").Return(errors.New("disk full"))
	var out bytes.Buffer
	err := cmd.RunProfileRemove(m, "ro", &out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save")
}

// --- runProfileEdit ---

func TestRunProfileEdit_KeepCurrentWhenBlank(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := &config.Profile{
		Name:      "ro",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}
	m.On("GetProfile", "ro").Return(p, nil)
	m.On("GetDatabases").Return([]config.Database{{Name: "db1", Type: config.DBTypeSQLite}})
	// When input is blank, profile should be unchanged
	m.On("UpdateProfile", *p).Return(nil)
	m.On("Save").Return(nil)

	// Simulate blank input for both prompts: two blank tokens won't scan via Fscan,
	// so we send newlines which fmt.Fscan will skip. Use spaces to trigger blank detection.
	in := strings.NewReader(" \n \n")
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileEdit(m, "ro", in, &out))
}

func TestRunProfileEdit_UpdateDatabases(t *testing.T) {
	m := mocks.NewMockStorer(t)
	p := &config.Profile{
		Name:      "ro",
		Databases: []string{"db1"},
		Allow:     []config.Operation{config.OpSelect},
	}
	m.On("GetProfile", "ro").Return(p, nil)
	m.On("GetDatabases").Return([]config.Database{
		{Name: "db1", Type: config.DBTypeSQLite},
		{Name: "db2", Type: config.DBTypeSQLite},
	})
	m.On("UpdateProfile", config.Profile{
		Name:      "ro",
		Databases: []string{"db2"},
		Allow:     []config.Operation{config.OpSelect},
	}).Return(nil)
	m.On("Save").Return(nil)

	// First input = new dbs, second = blank ops (keep current)
	in := strings.NewReader("db2\n \n")
	var out bytes.Buffer
	require.NoError(t, cmd.RunProfileEdit(m, "ro", in, &out))
}
