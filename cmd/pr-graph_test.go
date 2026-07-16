package cmd

import (
	"testing"
	"time"
)

func TestParseUntil(t *testing.T) {
	// A date-only value is inclusive through the end of that UTC day.
	got, err := parseUntil("2026-07-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	if !got.Equal(want) {
		t.Errorf("date-only until = %v, want %v", got, want)
	}

	// A PR created later on the --until day must be included.
	created := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	if created.After(got) {
		t.Errorf("PR created at %v was excluded by --until 2026-07-15", created)
	}

	// A PR created on the next day must be excluded.
	nextDay := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	if !nextDay.After(got) {
		t.Errorf("PR created at %v should be excluded by --until 2026-07-15", nextDay)
	}

	// An RFC 3339 value is used as an exact instant, unchanged.
	got, err = parseUntil("2026-07-15T10:30:00+09:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = time.Date(2026, 7, 15, 10, 30, 0, 0, time.FixedZone("", 9*3600))
	if !got.Equal(want) {
		t.Errorf("RFC3339 until = %v, want %v", got, want)
	}
}
