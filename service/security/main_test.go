package security

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Inject test value instead of loading from .env
	var (
		SecretKey              = []byte("SOME-SECRET-KEY")
		TokenExpiration        = 60   // 60 minutes = 1 hour
		RefreshTokenExpiration = 1440 // 1440 minutes = 1 day
	)
	service = NewJWTService(SecretKey, time.Duration(TokenExpiration), time.Duration(RefreshTokenExpiration))
	os.Exit(m.Run())
}
