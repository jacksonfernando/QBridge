package config_test

import (
	"testing"

	"github.com/jacksonfernando/qbridge/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecrypt_TooShortData(t *testing.T) {
	_, err := config.Decrypt([]byte("short"), "pw")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecrypt_CorruptCiphertext(t *testing.T) {
	enc, err := config.Encrypt([]byte("hello"), "pw")
	require.NoError(t, err)
	// Flip a byte in the ciphertext region (after the 16-byte salt).
	enc[len(enc)-1] ^= 0xFF
	_, err = config.Decrypt(enc, "pw")
	assert.Error(t, err)
}

func TestEncrypt_LargePlaintext(t *testing.T) {
	large := make([]byte, 1<<20) // 1 MB
	for i := range large {
		large[i] = byte(i % 256)
	}
	enc, err := config.Encrypt(large, "password")
	require.NoError(t, err)

	dec, err := config.Decrypt(enc, "password")
	require.NoError(t, err)
	assert.Equal(t, large, dec)
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	enc, err := config.Encrypt([]byte{}, "pw")
	require.NoError(t, err)
	dec, err := config.Decrypt(enc, "pw")
	require.NoError(t, err)
	assert.Empty(t, dec)
}
