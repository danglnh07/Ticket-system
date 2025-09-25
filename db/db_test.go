package db

import (
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var (
	dbConn    string
	redisConn string
	queries   *Queries
)

func TestMain(m *testing.M) {
	dbConn = os.Getenv("DB_CONN")
	redisConn = os.Getenv("REDIS_ADDRESS")

	queries = NewQueries()
	os.Exit(m.Run())
}

func TestDB(t *testing.T) {
	// Test connection
	err := queries.ConnectDB(dbConn)
	require.NoError(t, err)

	// Test auto migration
	err = queries.AutoMigration()
	require.NoError(t, err)
}

func TestCache(t *testing.T) {
	// Test connection
	err := queries.ConnectRedis(&redis.Options{
		Addr: redisConn,
	})
	require.NoError(t, err)

	// Try caching
	key := "some-key"
	str := "some-random-value"
	queries.Cache.Set(t.Context(), key, str, time.Second*5)
	val, err := queries.GetCache(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, str, val)

	// Sleep the main thread for 6 seconds to check if the expiration works
	time.Sleep(time.Second * 6)
	val, err = queries.GetCache(t.Context(), key)
	require.Error(t, err)
	require.ErrorContains(t, err, "cache miss")
	require.Empty(t, val)
}
