//go:build linux

package main

import (
	"syscall"
	"time"
)

// fileLastAccessTime extracts the access time from a syscall.Stat_t. The
// Linux stat layout names the field Atim, whereas Darwin calls it Atimespec.
func fileLastAccessTime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
}
