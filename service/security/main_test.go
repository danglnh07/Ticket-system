package security

import (
	"os"
	"testing"

	"github.com/danglnh07/ticket-system/util"
)

func TestMain(m *testing.M) {
	// Inject test value instead of loading from .env
	config := &util.Config{
		SecretKey:              []byte("SOME-SECRET-KEY"),
		TokenExpiration:        60,   // 60 minutes = 1 hour
		RefreshTokenExpiration: 1440, // 1440 minutes = 1 day
	}

	service = NewJWTService(config)
	os.Exit(m.Run())
}
