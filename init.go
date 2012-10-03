package main

import (
	"github.com/rcrowley/go-metrics"
)

var (
	findAttempts      = metrics.NewCounter()
	findSuccesses     = metrics.NewCounter()
	findFailures      = metrics.NewCounter()
	findTimer         = metrics.NewTimer()
	feedExecuteTiming = metrics.NewTimer()
)

func init() {
	registry.Register("feed_retriever_find_attempts", findAttempts)
	registry.Register("feed_retriever_find_successes", findSuccesses)
	registry.Register("feed_retriever_find_failures", findFailures)
	registry.Register("feed_retriever_find_timing", findTimer)
	registry.Register("frontend_user_feed_execute_timing", feedExecuteTiming)
}
