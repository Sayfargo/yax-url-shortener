package config

import (
	"testing"

	config_db_postgres "github.com/Sayfargo/yax-url-shortener/internal/config/db/postgres"
	config_storage "github.com/Sayfargo/yax-url-shortener/internal/config/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_StorageType(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want StorageType
	}{
		{
			name: "StorageTypeDB",
			cfg: &Config{
				DB:          &config_db_postgres.Config{DBDsn: "postgres://localhost:5432/db"},
				FileStorage: &config_storage.Config{FilePath: "/tmp/storage.json"},
			},
			want: StorageTypeDB,
		},
		{
			name: "StorageTypeFile",
			cfg: &Config{
				DB:          &config_db_postgres.Config{DBDsn: ""},
				FileStorage: &config_storage.Config{FilePath: "/tmp/storage.json"},
			},
			want: StorageTypeFile,
		},
		{
			name: "StorageTypeMemory",
			cfg: &Config{
				DB:          &config_db_postgres.Config{DBDsn: ""},
				FileStorage: &config_storage.Config{FilePath: ""},
			},
			want: StorageTypeMemory,
		},
		{
			name: "StorageTypeMemory",
			cfg: &Config{
				DB:          nil,
				FileStorage: nil,
			},
			want: StorageTypeMemory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.StorageType()
			assert.Equal(t, tt.want, got)

		})
	}
}

func TestConfigLoad_WithFlags(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedDsn := "postgres://user:pass@localhost:5432/db"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("FILE_STORAGE_PATH", "")
	t.Setenv("DATABASE_DSN", "")

	args := []string{
		"-a", expectedAddr,
		"-b", expectedBaseURL,
		"-d", expectedDsn,
		"-f", expectedFileStoragePath,
	}
	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedDsn, config.DB.DBDsn)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

func TestConfigLoad_WithEnv(t *testing.T) {

	expectedAddr := "localhost:9090"
	expectedBaseURL := "http://localhost:9090"
	expectedDsn := "postgres://user:pass@localhost:5432/db"
	expectedFileStoragePath := "./urls.json"

	t.Setenv("SERVER_ADDRESS", expectedAddr)
	t.Setenv("BASE_URL", expectedBaseURL)
	t.Setenv("DATABASE_DSN", expectedDsn)
	t.Setenv("FILE_STORAGE_PATH", expectedFileStoragePath)

	args := []string{}
	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedDsn, config.DB.DBDsn)
	assert.Equal(t, expectedFileStoragePath, config.FileStorage.FilePath)

}

func TestConfigLoad_WithDefaultParams(t *testing.T) {
	expectedAddr := "localhost:8080"
	expectedBaseURL := "http://localhost:8080"
	expectedDsn := ""
	expectedBaseFileStoragePath := ""

	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("FILE_STORAGE_PATH", "")

	args := []string{}

	config, err := Load(args)
	require.NoError(t, err)

	assert.Equal(t, expectedAddr, config.Server.Addr)
	assert.Equal(t, expectedBaseURL, config.Server.BaseURL)
	assert.Equal(t, expectedDsn, config.DB.DBDsn)
	assert.Equal(t, expectedBaseFileStoragePath, config.FileStorage.FilePath)
}
