package scheduler

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCronJob(t *testing.T) {
	scheduler := NewScheduler()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	count := 0
	scheduler.AddJob("@every 3s", func() {
		// Add the current time to the timestamp
		logger.Info("", "time", time.Now())
		count++
	})

	scheduler.RunCronJobs()

	// Len the cron run 10 times
	time.Sleep(time.Second * 32) // Allow for some delay, but the job cannot run to 11 times

	require.Equal(t, 10, count)
}
