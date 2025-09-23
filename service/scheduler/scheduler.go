package scheduler

import "github.com/robfig/cron/v3"

type Scheduler struct {
	c *cron.Cron
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		c: cron.New(cron.WithSeconds()),
	}
}

func (scheduler *Scheduler) AddJob(schedule string, job cron.FuncJob) error {
	_, err := scheduler.c.AddFunc(schedule, job)
	return err
}

func (scheduler *Scheduler) RunCronJobs() {
	scheduler.c.Start()
}
