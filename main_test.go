package utils_pkg

import (
	"regexp"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version() != "v0.6.0" {
		t.Fatalf("Version() = %q, want v0.6.0", Version())
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(Version()) {
		t.Fatalf("Version() is not semver: %q", Version())
	}
}
