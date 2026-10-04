//go:build linux

package main

import (
	"os"
	"strings"
)

// defaultRouteInterface returns the interface the kernel would use for the
// default route, read from /proc/net/route. A missing file or no default route
// leaves an empty result so the physical carrier's rates are kept.
func defaultRouteInterface() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 {
			continue // header row
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Column 1 is the destination in network-byte-order hex; 00000000 is default.
		if fields[1] == "00000000" {
			return fields[0]
		}
	}
	return ""
}
