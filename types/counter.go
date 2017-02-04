package types

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Counter struct {
	total     int64
	current   int64
	startTime time.Time
	stopTime  time.Time
	Interval  time.Duration
	stop      chan struct{}
}

func (c *Counter) Total() int64 {
	return c.total
}

func (c *Counter) Seconds() float64 {
	return c.stopTime.Sub(c.startTime).Seconds()
}

func (c *Counter) Avg() float64 {
	return float64(c.total) / c.Seconds()
}

func (c *Counter) Add() {
	atomic.AddInt64(&c.total, 1)
	atomic.AddInt64(&c.current, 1)
}

func (c *Counter) Stop() {
	c.stop <- struct{}{}
	c.stopTime = time.Now()
}

func (c *Counter) Start() {
	if c.stop == nil {
		c.stop = make(chan struct{})
	}
	c.startTime = time.Now()
	tick := time.Tick(c.Interval)
	for {
		select {
		case <-c.stop:
			return
		case <-tick:
			lastSecond := atomic.SwapInt64(&c.current, 0)
			fmt.Println(lastSecond)
		}
	}
}
