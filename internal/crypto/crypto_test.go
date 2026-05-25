package crypto

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", name)
}

func TestDeriveKey_Deterministic(t *testing.T) {
	keyPath := testdataPath("test_key_ed25519")

	key1, err := DeriveKey(keyPath, nil)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	if len(key1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key1))
	}

	key2, err := DeriveKey(keyPath, nil)
	if err != nil {
		t.Fatalf("DeriveKey second call failed: %v", err)
	}

	if string(key1) != string(key2) {
		t.Fatal("DeriveKey not deterministic: two calls produced different keys")
	}
}

func TestDeriveKey_InvalidPath(t *testing.T) {
	_, err := DeriveKey("/nonexistent/path", nil)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("my-secret-password")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	ciphertext, err := Encrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("same-input")

	c1, _ := Encrypt(key, plaintext)
	c2, _ := Encrypt(key, plaintext)

	if string(c1) == string(c2) {
		t.Fatal("two encryptions of same plaintext should produce different ciphertext (random nonce)")
	}
}
