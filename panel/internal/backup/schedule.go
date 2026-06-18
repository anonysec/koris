package backup

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"
)

// Schedule represents a parsed backup schedule configuration.
type Schedule struct {
	Type    string       // "daily", "weekly", "disabled"
	Hour    int          // 0-23
	Weekday time.Weekday // only for weekly
}

// ParseSchedule parses a schedule string like "daily:02", "weekly:sun:02", or "disabled".
func ParseSchedule(value string) Schedule {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(value)), ":")

	if len(parts) == 0 || parts[0] == "disabled" || parts[0] == "" {
		return Schedule{Type: "disabled"}
	}

	switch parts[0] {
	case "daily":
		hour := 2 // default
		if len(parts) >= 2 {
			if h, err := strconv.Atoi(parts[1]); err == nil && h >= 0 && h <= 23 {
				hour = h
			}
		}
		return Schedule{Type: "daily", Hour: hour}
	case "weekly":
		day := time.Sunday
		hour := 2
		if len(parts) >= 2 {
			day = parseWeekday(parts[1])
		}
		if len(parts) >= 3 {
			if h, err := strconv.Atoi(parts[2]); err == nil && h >= 0 && h <= 23 {
				hour = h
			}
		}
		return Schedule{Type: "weekly", Hour: hour, Weekday: day}
	default:
		return Schedule{Type: "disabled"}
	}
}

// parseWeekday converts a short day name to time.Weekday.
func parseWeekday(s string) time.Weekday {
	switch strings.ToLower(s) {
	case "sun", "sunday":
		return time.Sunday
	case "mon", "monday":
		return time.Monday
	case "tue", "tuesday":
		return time.Tuesday
	case "wed", "wednesday":
		return time.Wednesday
	case "thu", "thursday":
		return time.Thursday
	case "fri", "friday":
		return time.Friday
	case "sat", "saturday":
		return time.Saturday
	default:
		return time.Sunday
	}
}

// ShouldRun evaluates whether the current time matches the schedule pattern.
func ShouldRun(sched Schedule, now time.Time) bool {
	if sched.Type == "disabled" {
		return false
	}
	if now.Minute() != 0 {
		return false
	}
	if now.Hour() != sched.Hour {
		return false
	}
	if sched.Type == "weekly" && now.Weekday() != sched.Weekday {
		return false
	}
	return true
}

// StartScheduler begins a background goroutine that checks the schedule every minute.
func (s *Service) StartScheduler() {
	go func() {
		var lastTrigger time.Time
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			sched := ParseSchedule(s.cfg.Schedule)
			now := time.Now()

			if !ShouldRun(sched, now) {
				continue
			}

			// Prevent double-trigger in same hour
			if now.Truncate(time.Hour).Equal(lastTrigger.Truncate(time.Hour)) {
				continue
			}

			lastTrigger = now
			log.Println("[backup] scheduled backup starting")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			if _, err := s.CreateBackup(ctx, "scheduled"); err != nil {
				log.Printf("[backup] scheduled backup failed: %v", err)
			} else {
				log.Println("[backup] scheduled backup completed")
			}
			cancel()
		}
	}()
	log.Println("[backup] scheduler started")
}
