package url

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func cleanDB(t *testing.T) {
	t.Helper()

	_, err := testPool.Exec(context.Background(), `TRUNCATE shortened_urls RESTART IDENTITY`)
	require.NoError(t, err)
}

func newRepo(t *testing.T) *PostgresRepository {
	t.Helper()

	return NewPgRepo(testPool)
}
