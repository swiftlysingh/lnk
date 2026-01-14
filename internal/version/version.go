// Package version provides build version information.
package version

import (
	"fmt"
	"runtime"
)

// Build information set by ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns a formatted version string.
func Info() string {
	return fmt.Sprintf("lnk %s (%s) built on %s with %s",
		Version, Commit, Date, runtime.Version())
}

// Short returns just the version number.
func Short() string {
	return Version
}
