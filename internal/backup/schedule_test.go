package backup

import (
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTyp string
		wantHr  int
		wantDay time.Weekday
	}{
		{
			name:    "valid daily",
			input:   "daily:02",
			wantTyp: "daily",
			wantHr:  2,
		},
		{
			name:    "valid daily hour 23",
			input:   "daily:23",
			wantTyp: "daily",
			wantHr:  23,
		},
		{
			name:    "valid weekly sunday",
			input:   "weekly:sun:03",
			wantTyp: "weekly",
			wantHr:  3,
			wantDay: time.Sunday,
		},
		{
			name:    "valid weekly friday",
			input:   "weekly:fri:18",
			wantTyp: "weekly",
			wantHr:  18,
			wantDay: time.Friday,
		},
		{
			name:    "disabled",
			input:   "disabled",
			wantTyp: "disabled",
		},
		{
			name:    "empty string",
			input:   "",
			wantTyp: "disabled",
		},
		{
			name:    "invalid format monthly",
			input:   "monthly:01",
			wantTyp: "disabled",
		},
		{
			name:    "invalid hour 25 falls back to default",
			input:   "daily:25",
			wantTyp: "daily",
			wantHr:  2, // default hour when invalid
		},
		{
			name:    "weekly with invalid day defaults to sunday",
			input:   "weekly:xyz:02",
			wantTyp: "weekly",
			wantHr:  2,
			wantDay: time.Sunday,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSchedule(tt.input)
			if got.Type != tt.wantTyp {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantTyp)
			}
			if got.Hour != tt.wantHr {
				t.Errorf("Hour = %d, want %d", got.Hour, tt.wantHr)
			}
			if tt.wantTyp == "weekly" && got.Weekday != tt.wantDay {
				t.Errorf("Weekday = %v, want %v", got.Weekday, tt.wantDay)
			}
			// Invariant: hour always in [0, 23]
			if got.Hour < 0 || got.Hour > 23 {
				t.Errorf("Hour %d out of valid range [0, 23]", got.Hour)
			}
		})
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name  string
		sched Schedule
		now   time.Time
		want  bool
	}{
		{
			name:  "disabled always false",
			sched: Schedule{Type: "disabled"},
			now:   time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC),
			want:  false,
		},
		{
			name:  "daily match",
			sched: Schedule{Type: "daily", Hour: 2},
			now:   time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC), // Tuesday 02:00
			want:  true,
		},
		{
			name:  "daily hour mismatch",
			sched: Schedule{Type: "daily", Hour: 2},
			now:   time.Date(2024, 1, 15, 5, 0, 0, 0, time.UTC),
			want:  false,
		},
		{
			name:  "daily non-zero minute",
			sched: Schedule{Type: "daily", Hour: 2},
			now:   time.Date(2024, 1, 15, 2, 30, 0, 0, time.UTC),
			want:  false,
		},
		{
			name:  "weekly match",
			sched: Schedule{Type: "weekly", Hour: 3, Weekday: time.Sunday},
			now:   time.Date(2024, 1, 14, 3, 0, 0, 0, time.UTC), // Sunday 03:00
			want:  true,
		},
		{
			name:  "weekly wrong day",
			sched: Schedule{Type: "weekly", Hour: 3, Weekday: time.Sunday},
			now:   time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC), // Monday 03:00
			want:  false,
		},
		{
			name:  "weekly non-zero minute",
			sched: Schedule{Type: "weekly", Hour: 3, Weekday: time.Sunday},
			now:   time.Date(2024, 1, 14, 3, 15, 0, 0, time.UTC), // Sunday 03:15
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRun(tt.sched, tt.now)
			if got != tt.want {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}
