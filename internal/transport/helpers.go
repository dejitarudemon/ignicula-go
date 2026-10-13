package transport

import "time"

func deadline(timeout time.Duration) time.Time {
	return time.Now().Add(timeout)
}
