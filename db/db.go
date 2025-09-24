package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// The queries object for interacting with database and cache
type Queries struct {
	DB    *gorm.DB
	Cache *redis.Client
}

// Constructor for Queries
func NewQueries() *Queries {
	return &Queries{}
}

// Connect to Postgres
func (queries *Queries) ConnectDB(connStr string) error {
	conn, err := gorm.Open(postgres.Open(connStr))
	if err != nil {
		return err
	}

	queries.DB = conn
	return nil
}

// Run postgres database auto migration
func (queries *Queries) AutoMigration() error {
	return queries.DB.AutoMigrate(&Account{}, &Membership{}, &Event{}, &Ticket{}, &Booking{})
}

// Connect to Redis
func (queries *Queries) ConnectRedis(opt *redis.Options) error {
	queries.Cache = redis.NewClient(opt)
	_, err := queries.Cache.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	return nil
}

// Set cache value. If expired = 0, it will set the expiration time to 1 hour instead of no expiration
func (queries *Queries) SetCache(ctx context.Context, key string, val string, expired time.Duration) {
	if expired == 0 {
		expired = time.Hour
	}
	queries.Cache.Set(ctx, key, val, expired)
}

// Custom error: if redis failed to get cached value
type RedisInternalErr struct {
	Message string
	Err     error
}

func (err *RedisInternalErr) Error() string {
	return fmt.Sprintf("%s: %v", err.Message, err.Err)
}

// Custome error: if key doesn't exist in cache
type RedisNoValueErr struct {
	Message string
}

func (err *RedisNoValueErr) Error() string {
	return err.Message
}

// Get cache value
func (queries *Queries) GetCache(ctx context.Context, key string) (string, error) {
	val, err := queries.Cache.Get(ctx, key).Result()

	// If actually found value, return the val
	if err == nil {
		return val, nil
	}

	// If redis error
	if err != redis.Nil {
		return "", &RedisInternalErr{Message: "error getting value from redis cache", Err: err}
	}

	// If the value of the key simply don't exists, or expired
	return "", &RedisNoValueErr{"key not exists, or key-value expired"}
}
