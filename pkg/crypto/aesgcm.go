package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

func EncryptAESGCM(text string, keyStr string) (string, error) {

	aesblock, err := aes.NewCipher([]byte(keyStr))
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", fmt.Errorf("cipher new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("io read full: %w", err)
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(text), nil)

	return hex.EncodeToString(cipherText), nil
}

func DecryptAESGCM(encryptedHex string, keyStr string) (string, error) {

	cipherText, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", fmt.Errorf("hex decode string: %w", err)
	}

	aesblock, err := aes.NewCipher([]byte(keyStr))
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", fmt.Errorf("cipher new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherText) < nonceSize {
		return "", errors.New("err open")
	}

	nonce, actual := cipherText[:nonceSize], cipherText[nonceSize:]

	text, err := aesGCM.Open(nil, nonce, actual, nil)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}

	return string(text), nil
}
