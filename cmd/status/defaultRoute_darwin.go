//go:build darwin

package main

import (
	"context"
	"strings"
	"time"
)

// defaultRouteInterface returns the interface for the IPv4 default route via
// `route -n get default`, matching the upstream Mole probe.
func defaultRouteInterface() string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	out, err := runCmd(ctx, "route", "-n", "get", "default")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(key) == "interface" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
