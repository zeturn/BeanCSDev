package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	EncryptString(plaintext string) ([]byte, error)
	DecryptString(ciphertext []byte) (string, error)
}

type AESGCMCipher struct {
	aead cipher.AEAD
}

func NewAESGCMCipher(hexKey string) (*AESGCMCipher, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes encoded as hex")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESGCMCipher{aead: aead}, nil
}

func (c *AESGCMCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := c.aead.Seal(nonce, nonce, plaintext, nil)
	return out, nil
}

func (c *AESGCMCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:c.aead.NonceSize()]
	payload := ciphertext[c.aead.NonceSize():]
	return c.aead.Open(nil, nonce, payload, nil)
}

func (c *AESGCMCipher) EncryptString(plaintext string) ([]byte, error) {
	return c.Encrypt([]byte(plaintext))
}

func (c *AESGCMCipher) DecryptString(ciphertext []byte) (string, error) {
	out, err := c.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
