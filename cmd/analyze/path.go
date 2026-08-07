package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// validatePath guards every path that reaches the scanner or the open/preview
// helpers. It rejects relative paths, null bytes, and traversal components so a
// hostile argument cannot escape the intended scan root or reach files it must
// not.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null bytes")
	}
	// Check for path traversal attempts (.. components).
	if slices.Contains(strings.Split(path, string(filepath.Separator)), "..") {
		return fmt.Errorf("path contains traversal components: %s", path)
	}
	return nil
}
