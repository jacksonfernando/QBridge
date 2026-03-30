package config_test

import (
	"testing"

	"github.com/jacksonfernando/qbridge/internal/config"
)

func TestEncryptDecrypt(t *testing.T) {
	plain := []byte("super secret database password")
	password := "master-password-123"

	enc, err := config.Encrypt(plain, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	dec, err := config.Decrypt(enc, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(dec) != string(plain) {
		t.Errorf("decrypted %q != original %q", dec, plain)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	enc, _ := config.Encrypt([]byte("data"), "correct")
	_, err := config.Decrypt(enc, "wrong")
	if err == nil {
		t.Fatal("expected error decrypting with wrong password, got nil")
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	plain := []byte("hello")
	password := "pw"
	enc1, _ := config.Encrypt(plain, password)
	enc2, _ := config.Encrypt(plain, password)
	if string(enc1) == string(enc2) {
		t.Error("two encryptions of same plaintext should differ (different nonces/salts)")
	}
}
