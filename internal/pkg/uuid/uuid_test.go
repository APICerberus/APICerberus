package uuid

import (
	"regexp"
	"testing"
)

func TestNewStringFormat(t *testing.T) {
	t.Parallel()

	id, err := NewString()
	if err != nil {
		t.Fatalf("NewString error: %v", err)
	}

	re := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	if !re.MatchString(id) {
		t.Fatalf("invalid uuid format: %q", id)
	}
}

func TestNewStringUniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id, err := NewString()
		if err != nil {
			t.Fatalf("NewString error: %v", err)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate uuid generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}
