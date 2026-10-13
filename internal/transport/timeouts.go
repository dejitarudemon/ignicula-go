package transport

import "time"

type Timeouts struct {
	Read  time.Duration
	Write time.Duration
}
