package messages

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestSyncReadinessReportsProbeResult(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query syncQuery
		want  bool
	}{
		{"schema ready", syncQuery{contains: "shop_message_counters", rows: [][]driver.Value{{true}}}, true},
		{"schema not ready", syncQuery{contains: "shop_message_counters", rows: [][]driver.Value{{false}}}, false},
		{"probe error fails closed", syncQuery{contains: "shop_message_counters", err: errors.New("relation does not exist")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := syncRepo(t, tc.query)
			if got := SyncReadiness(repo.db, true)(context.Background()); got != tc.want {
				t.Fatalf("got %t want %t", got, tc.want)
			}
		})
	}
}

func TestSyncReadinessFlagOffSkipsProbe(t *testing.T) {
	// The single scripted result must still be unconsumed after the flag-off
	// call; the enabled call afterwards consumes it, and syncRepo's cleanup
	// fails the test if it is left over.
	repo, _ := syncRepo(t, syncQuery{contains: "shop_message_counters", rows: [][]driver.Value{{true}}})
	if SyncReadiness(repo.db, false)(context.Background()) {
		t.Fatal("flag off must report false")
	}
	if got := SyncReadiness(repo.db, true)(context.Background()); !got {
		t.Fatal("scripted probe query must still be available, proving the flag-off call issued none")
	}
}
