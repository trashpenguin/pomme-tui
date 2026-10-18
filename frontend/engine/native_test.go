package engine

import (
	"testing"
	"time"
)

func TestCatalogActionsHaveEnoughTimeToComplete(t *testing.T) {
	if got := methodTimeout("save_item"); got <= 30*time.Second {
		t.Fatalf("save_item timeout=%s, backend allows 30 seconds", got)
	}
	if got := methodTimeout("save_playlist"); got <= 30*time.Second {
		t.Fatalf("save_playlist timeout=%s, want time for a multi-track save", got)
	}
}
