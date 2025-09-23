package security

import (
	"testing"

	"github.com/danglnh07/ticket-system/ticket-system/util"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	// Create test data
	str := util.RandomString(10)

	// Try encoding and decode
	encode := Encode(str)
	require.NotEmpty(t, encode)
	require.Equal(t, str, Decode(encode))
}

func TestBcryptHash(t *testing.T) {
	// Create test data
	str := util.RandomString(10)

	// Bcrypt hash
	hashed, err := BcryptHash(str)
	require.NoError(t, err)

	// Compare hash and raw string
	require.Equal(t, true, BcryptCompare(hashed, str))
}
