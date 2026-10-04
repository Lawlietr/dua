//go:build linux

package main

import (
	"os"
	"strings"
	"testing"
)

// TestDefaultRouteInterfaceMatchesKernelDefault computes the default interface
// independently from /proc/net/route and checks defaultRouteInterface agrees,
// so a change to the parser is caught on Linux CI.
func TestDefaultRouteInterfaceMatchesKernelDefault(t *testing.T) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		t.Skipf("no /proc/net/route: %v", err)
	}
	want := ""
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 {
			continue // header row
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "00000000" {
			want = fields[0]
			break
		}
	}
	if want == "" {
		t.Skip("host has no default route")
	}
	if got := defaultRouteInterface(); got != want {
		t.Fatalf("defaultRouteInterface() = %q, want %q", got, want)
	}
}
