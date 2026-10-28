package main

import (
	"testing"
	"time"
)

func TestOperatingHours(t *testing.T) {
	start := time.Date(2026, 11, 2, 9, 0, 0, 0, india)
	for _, tt := range []struct {
		name       string
		start, end time.Time
		ok         bool
	}{{"within shift", start, start.Add(time.Hour), true}, {"ends at closing", start, start.Add(9 * time.Hour), true}, {"after closing", start, start.Add(9*time.Hour + time.Minute), false}, {"overnight", start, start.Add(24 * time.Hour), false}, {"before shift", start.Add(-time.Minute), start.Add(time.Hour), false}} {
		t.Run(tt.name, func(t *testing.T) {
			if got := withinHours(tt.start, tt.end, 9, 18); got != tt.ok {
				t.Fatalf("got %v want %v", got, tt.ok)
			}
		})
	}
}
func TestWeekStartsMondayInIndia(t *testing.T) {
	sunday := time.Date(2026, 11, 8, 23, 0, 0, 0, india)
	if got := weekStart(sunday); got.Weekday() != time.Monday || got.Day() != 2 || got.Hour() != 0 {
		t.Fatalf("bad week boundary: %v", got)
	}
}
func TestDurationAndDateLimits(t *testing.T) {
	p := resourcePolicy{Max: 4, Open: 9, Close: 18, From: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)}
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, india)
	if e := checkTime(p, start, start.Add(5*time.Hour)); e == nil || e.Error() != "duration_limit" {
		t.Fatal("long booking accepted")
	}
	if e := checkTime(p, start.AddDate(0, 1, 0), start.AddDate(0, 1, 0).Add(time.Hour)); e == nil || e.Error() != "outside_dates" {
		t.Fatal("out of range accepted")
	}
}
