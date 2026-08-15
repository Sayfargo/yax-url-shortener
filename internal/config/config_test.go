package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoad_WithFlags(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("FILE_STORAGE_PATH", "")

	args := []string{
		"-a", expectedAddr,
		"-b", expectedBaseURL,
		"-f", expectedFileStoragePath,
	}
	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

func TestConfigLoad_WithEnv(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", expectedAddr)
	t.Setenv("BASE_URL", expectedBaseURL)
	t.Setenv("FILE_STORAGE_PATH", expectedFileStoragePath)

	args := []string{}
	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

func TestConfigLoad_WithDefaultParams(t *testing.T) {
	expectedAddr := "localhost:8080"
	expectedBaseURL := "http://localhost:8080"
	expectedBaseFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("FILE_STORAGE_PATH", "")

	args := []string{}

	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedBaseFileStoragePath, config.FileStorage.FilePath)
}
