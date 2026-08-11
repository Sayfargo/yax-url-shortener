package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
func TestConfigLoad_WithFlags(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASIC_URL", "")
	t.Setenv("FILE_STORAGE_PATH", "")

	args := []string{
		"-a", expectedAddr,
		"-b", expectedBaseURL,
		"-f", expectedFileStoragePath,
	}
	config := Load(args)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

// Если указана переменная окружения, то используется она.
func TestConfigLoad_WithEnv(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", expectedAddr)
	t.Setenv("BASIC_URL", expectedBaseURL)
	t.Setenv("FILE_STORAGE_PATH", expectedFileStoragePath)

	args := []string{}
	config := Load(args)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

// Если нет ни переменной окружения, ни флага, то используется значение по умолчанию.
// Тест может упасть если поменяются значения по умолчанию в коде
func TestConfigLoad_WithDefaultParams(t *testing.T) {
	expectedAddr := "localhost:8080"
	expectedBaseURL := "http://localhost:8080"
	expectedBaseFileStoragePath := "./urls.json"

	args := []string{}

	config := Load(args)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedBaseFileStoragePath, config.FileStorage.FilePath)
}
