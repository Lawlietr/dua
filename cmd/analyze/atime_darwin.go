//go:build darwin

package main

import (
	"syscall"
	"time"
)

// fileLastAccessTime extracts the access time from a syscall.Stat_t. The
// Darwin stat layout names the field Atimespec, whereas Linux calls it Atim.
func fileLastAccessTime(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
}
