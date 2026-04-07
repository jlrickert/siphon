package siphon

import "fmt"

// CronConfig holds the parameters needed to generate a cron entry.
type CronConfig struct {
	Minute  string // cron minute field (e.g., "0", "*/15")
	Hour    string // cron hour field (e.g., "2", "*")
	Day     string // day of month ("*" if empty)
	Month   string // month ("*" if empty)
	Weekday string // day of week ("*" if empty)
	Command string // the full command to run
}

// GenerateCronEntry returns a single cron line: "M H D Mo W command".
func GenerateCronEntry(cfg CronConfig) string {
	day := cfg.Day
	if day == "" {
		day = "*"
	}
	month := cfg.Month
	if month == "" {
		month = "*"
	}
	weekday := cfg.Weekday
	if weekday == "" {
		weekday = "*"
	}

	return fmt.Sprintf("%s %s %s %s %s %s", cfg.Minute, cfg.Hour, day, month, weekday, cfg.Command)
}
