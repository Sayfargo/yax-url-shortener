// Модуль crypto предоставляет криптографические утилиты для защиты данных.
// Пакет реализует симметричное шифрование и дешифрование строк с использованием
// алгоритма AES в режиме GCM (Galois/Counter Mode), обеспечивающем аутентифицированное шифрование.
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

// EncryptAESGCM шифрует исходную строку [text] с использованием ключа [keyStr]
// по алгоритму AES-GCM.
//
// Метод автоматически генерирует криптографически стойкую случайную последовательность (nonce)
// с помощью [rand.Reader] для каждой операции шифрования. Полученный nonce прикрепляется
// в качестве префикса к зашифрованным данным, после чего весь результат кодируется в шестнадцатеричную строку (hex).
//
// Возвращает зашифрованную hex-строку или обернутую ошибку при сбое инициализации шифра или чтения энтропии.
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

// DecryptAESGCM расшифровывает шестнадцатеричную строку [encryptedHex], зашифрованную ранее
// методом [EncryptAESGCM], используя тот же ключ [keyStr].
//
// Метод сначала декодирует hex-строку, извлекает nonce из префикса полученного байтового слайса,
// проверяет целостность данных и выполняет дешифрование оставшейся части сообщения.
//
// Возвращает оригинальную строку или ошибку, если строка повреждена, ключ не совпадает
// или аутентификация данных (интегрити-чек GCM) завершилась неудачно.
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
