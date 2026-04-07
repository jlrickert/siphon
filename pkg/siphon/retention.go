package siphon

import (
	"sort"
	"time"
)

// RetentionPolicy defines how many and how old backups to keep.
type RetentionPolicy struct {
	KeepCount *int    `yaml:"keep_count,omitempty" json:"keep_count,omitempty"`
	KeepAge   *string `yaml:"keep_age,omitempty" json:"keep_age,omitempty"` // duration string like "720h" or "30d"
}

// ParseKeepAge parses the KeepAge field into a time.Duration. It supports Go
// duration strings (e.g., "720h") and a convenience "d" suffix for days
// (e.g., "30d" -> 720h).
func (rp *RetentionPolicy) ParseKeepAge() (time.Duration, error) {
	if rp.KeepAge == nil || *rp.KeepAge == "" {
		return 0, nil
	}
	return ParseDuration(*rp.KeepAge)
}

// ParseDuration parses a duration string with support for a "d" suffix
// for days (e.g., "30d" -> 720h0m0s).
func ParseDuration(s string) (time.Duration, error) {
	if len(s) > 0 && s[len(s)-1] == 'd' {
		// Parse the numeric prefix as days.
		days := s[:len(s)-1]
		d, err := time.ParseDuration(days + "h")
		if err != nil {
			return 0, err
		}
		return d * 24, nil
	}
	return time.ParseDuration(s)
}

// ApplyRetention returns the list of backup descriptors that should be pruned
// according to the given retention policy. Backups are sorted newest-first;
// the newest backups are kept. This is a dry-run operation -- it does not
// delete anything.
func ApplyRetention(backups []BackupDescriptor, policy RetentionPolicy, now time.Time) []BackupDescriptor {
	if len(backups) == 0 {
		return nil
	}

	// Sort newest first.
	sorted := make([]BackupDescriptor, len(backups))
	copy(sorted, backups)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	// Track which indices to prune.
	prune := make(map[int]bool)

	// Count-based pruning: keep the N most recent.
	if policy.KeepCount != nil && *policy.KeepCount > 0 {
		count := *policy.KeepCount
		for i := count; i < len(sorted); i++ {
			prune[i] = true
		}
	}

	// Age-based pruning: prune backups older than the threshold.
	if policy.KeepAge != nil && *policy.KeepAge != "" {
		maxAge, err := ParseDuration(*policy.KeepAge)
		if err == nil && maxAge > 0 {
			cutoff := now.Add(-maxAge)
			for i, b := range sorted {
				if b.Timestamp.Before(cutoff) {
					prune[i] = true
				}
			}
		}
	}

	var result []BackupDescriptor
	for i, b := range sorted {
		if prune[i] {
			result = append(result, b)
		}
	}

	return result
}
