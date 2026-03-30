package policy_test

import (
	"testing"

	"github.com/jacksonfernando/qbridge/internal/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- firstKeyword edge cases (tested through ClassifySQL) ---

func TestClassifySQL_UnclosedBlockComment(t *testing.T) {
	// An unclosed block comment means firstKeyword returns "" → error
	_, err := policy.ClassifySQL("/* no closing star-slash SELECT 1")
	assert.Error(t, err)
}

func TestClassifySQL_LineCommentNoNewline(t *testing.T) {
	// A line comment with no newline → no keyword found → error
	_, err := policy.ClassifySQL("-- only a comment")
	assert.Error(t, err)
}

func TestClassifySQL_MultipleLeadingComments(t *testing.T) {
	// Multiple block comments before keyword
	op, err := policy.ClassifySQL("/* a */ /* b */ SELECT 1")
	require.NoError(t, err)
	assert.Equal(t, "SELECT", string(op))
}

func TestClassifySQL_LineCommentFollowedByKeyword(t *testing.T) {
	op, err := policy.ClassifySQL("-- header\nUPDATE t SET x=1")
	require.NoError(t, err)
	assert.Equal(t, "UPDATE", string(op))
}

func TestClassifySQL_MixedCommentStyles(t *testing.T) {
	op, err := policy.ClassifySQL("/* block */ -- line\nDELETE FROM t")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", string(op))
}

func TestClassifySQL_RENAME(t *testing.T) {
	op, err := policy.ClassifySQL("RENAME TABLE old TO new")
	require.NoError(t, err)
	assert.Equal(t, "RENAME", string(op))
}

func TestClassifySQL_TRUNCATE(t *testing.T) {
	op, err := policy.ClassifySQL("TRUNCATE TABLE users")
	require.NoError(t, err)
	assert.Equal(t, "TRUNCATE", string(op))
}

// --- Check propagates ClassifySQL error ---

func TestCheck_UnknownStatement_ReturnsError(t *testing.T) {
	p := makeProfile("p", []string{"db"})
	err := policy.Check(p, "EXPLAIN SELECT 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy check failed")
}

func TestCheck_EmptySQL(t *testing.T) {
	p := makeProfile("p", []string{"db"})
	err := policy.Check(p, "")
	assert.Error(t, err)
}
