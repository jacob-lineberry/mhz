//go:build !windows

// Package cpu reports process CPU time.
package cpu

import (
	"syscall"
	"time"
)

// Time returns user+system CPU time used by this process so far.
func Time() time.Duration {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
