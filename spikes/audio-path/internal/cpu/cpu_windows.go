// Package cpu reports process CPU time.
package cpu

import (
	"time"

	"golang.org/x/sys/windows"
)

// Time returns user+system CPU time used by this process so far.
func Time() time.Duration {
	var c, e, k, u windows.Filetime
	_ = windows.GetProcessTimes(windows.CurrentProcess(), &c, &e, &k, &u)
	ft := func(f windows.Filetime) int64 { return int64(f.HighDateTime)<<32 | int64(f.LowDateTime) }
	return time.Duration((ft(k) + ft(u)) * 100)
}
