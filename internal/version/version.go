// Package version exposes build-time version information.
//
// The values are overridden at build time via -ldflags, e.g.:
//
//	go build -ldflags="-X audiosort/internal/version.Version=v1.2.3 \
//	    -X audiosort/internal/version.Commit=abc1234 \
//	    -X audiosort/internal/version.Date=2026-05-31"
package version

import "fmt"

// These variables are populated at build time with -ldflags -X.
// They keep sensible defaults so a plain `go build`/`go run` still works.
var (
	// Version is the semantic version of the build (e.g. "v1.2.3").
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// Date is the build timestamp (RFC3339 or YYYY-MM-DD).
	Date = "unknown"
)

// String returns a human-readable version string.
func String() string {
	if Commit == "none" {
		return Version
	}
	return fmt.Sprintf("%s (%s, built %s)", Version, Commit, Date)
}
