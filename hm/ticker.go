package hm

import "time"

func NewTicker(d time.Duration) chan time.Time {
	var (
		t = time.NewTicker(d)
		c = make(chan time.Time, 1)
	)
	go func() {
		for now := range t.C {
			c <- now
		}
	}()
	return c
}
