package shared

import (
	"testing"
	"time"
)

func TestParseServiceDate(t *testing.T) {
	for _, value := range []string{"0001-01-01", "2000-02-29", "2026-03-08", "2026-11-01", "9999-12-31"} {
		t.Run(value, func(t *testing.T) {
			got, err := ParseServiceDate(value)
			if err != nil || got.Format("2006-01-02") != value || got.Location() != time.UTC || got.Hour() != 0 {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
func TestParseServiceDateRejectsMalformed(t *testing.T) {
	for _, value := range []string{"", "2026-2-01", "2026-02-1", "2026-02-29", "1900-02-29", "2026-04-31", "2026-00-01", "2026-13-01", "2026-01-00", "0000-01-01", "10000-01-01", " 2026-01-01", "2026-01-01\n", "2026-01-01T00:00:00Z", "2026-01-01+03:00"} {
		t.Run(value, func(t *testing.T) {
			got, err := ParseServiceDate(value)
			if err == nil || !got.IsZero() {
				t.Fatalf("accepted %q: %v, %v", value, got, err)
			}
		})
	}
}
