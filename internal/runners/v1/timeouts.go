package runner_v1

import "time"

type Timeouts struct {
	Read   time.Duration
	Write  time.Duration
	Delete time.Duration
	Ping   time.Duration

	Backoff     time.Duration
	MaxAttempts int
}
