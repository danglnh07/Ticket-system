package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	conn    string
	queries *Queries
)

func TestMain(m *testing.M) {
	conn = os.Getenv("DB_CONN")
	queries = NewQueries()
	os.Exit(m.Run())
}

func TestDB(t *testing.T) {
	// Test connection
	err := queries.ConnectDB(conn)
	require.NoError(t, err)

	// Test auto migration
	err = queries.AutoMigration()
	require.NoError(t, err)
}
