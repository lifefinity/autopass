package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ssh"
)

var (
	hkdfSalt = []byte("autopass-salt-v1")
	hkdfInfo = []byte("autopass-v1")
)

func DeriveKey(sshKeyPath string, passphrase []byte) ([]byte, error) {
	keyData, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading SSH key: %w", err)
	}

	rawBytes, err := extractPrivateKeyBytes(keyData, passphrase)
	if err != nil {
		return nil, fmt.Errorf("parsing SSH key: %w", err)
	}

	hkdfReader := hkdf.New(sha256.New, rawBytes, hkdfSalt, hkdfInfo)
	derivedKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, fmt.Errorf("HKDF expansion: %w", err)
	}

	return derivedKey, nil
}

func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return plaintext, nil
}

func extractPrivateKeyBytes(pemData, passphrase []byte) ([]byte, error) {
	var rawKey interface{}
	var err error

	if passphrase != nil {
		rawKey, err = ssh.ParseRawPrivateKeyWithPassphrase(pemData, passphrase)
	} else {
		rawKey, err = ssh.ParseRawPrivateKey(pemData)
	}
	if err != nil {
		return nil, err
	}

	switch k := rawKey.(type) {
	case *ed25519.PrivateKey:
		return []byte(*k), nil
	case ed25519.PrivateKey:
		return []byte(k), nil
	default:
		block, _ := pem.Decode(pemData)
		if block == nil {
			return nil, fmt.Errorf("unsupported key type %T and no PEM block found", rawKey)
		}
		return block.Bytes, nil
	}
}
