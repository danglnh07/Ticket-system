package worker

import (
	"context"
	"testing"

	"github.com/danglnh07/ticket-system/ticket-system/util"
	"github.com/stretchr/testify/require"
)

func TestDistributeSendEmail(t *testing.T) {
	// Create Payload
	payload := SendVerifyEmailPayload{
		Email:    "test@gmail.com",
		Username: util.RandomString(7),
	}

	// Distribute task
	err := distributor.DistributeTaskSendVerifyEmail(context.Background(), payload)
	require.NoError(t, err)
}

func TestSendVerifyEmail(t *testing.T) {
	// Create Payload
	payload := SendVerifyEmailPayload{
		Email:    "test@gmail.com",
		Username: util.RandomString(7),
	}

	// Process send verify email
	err := processor.(*RedisTaskProcessor).SendVerifyEmail(payload)
	require.NoError(t, err)
}
