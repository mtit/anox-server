package registry

import (
	"regexp"
	"testing"
)

func TestGenerateInstanceIDUsesReadableTimestampSuffix(t *testing.T) {
	id := generateInstanceID("user-service")
	pattern := regexp.MustCompile(`^user-service-\d{14}[A-Za-z0-9]{6}$`)

	if !pattern.MatchString(id) {
		t.Fatalf("instance id %q does not match expected format", id)
	}
}
