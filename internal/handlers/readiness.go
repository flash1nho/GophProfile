package handlers

import "sync/atomic"

var shuttingDown atomic.Bool

func SetShuttingDown() {
	shuttingDown.Store(true)
}

func IsShuttingDown() bool {
	return shuttingDown.Load()
}
