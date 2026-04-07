package siphon_test

import (
	"testing"
	"time"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }

func makeBackups(count int, start time.Time) []siphon.BackupDescriptor {
	backups := make([]siphon.BackupDescriptor, count)
	for i := 0; i < count; i++ {
		backups[i] = siphon.BackupDescriptor{
			ID:        "backup-" + time.Duration(i).String(),
			Timestamp: start.Add(-time.Duration(i) * 24 * time.Hour),
		}
	}
	return backups
}

func TestApplyRetention_CountBased(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	backups := makeBackups(5, now)

	policy := siphon.RetentionPolicy{
		KeepCount: intPtr(3),
	}

	pruned := siphon.ApplyRetention(backups, policy, now)
	require.Len(t, pruned, 2, "should prune 2 of 5 backups when keeping 3")

	// Pruned should be the oldest two.
	for _, p := range pruned {
		require.True(t, p.Timestamp.Before(now.Add(-2*24*time.Hour)),
			"pruned backup should be older than the 3rd most recent")
	}
}

func TestApplyRetention_AgeBased(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	backups := makeBackups(5, now)
	// Backups: day 0 (now), day -1, day -2, day -3, day -4
	// Cutoff at 72h ago = day -3 exactly.
	// Before(cutoff) is strict: day -4 is before cutoff, day -3 equals cutoff.

	policy := siphon.RetentionPolicy{
		KeepAge: strPtr("72h"), // 3 days
	}

	pruned := siphon.ApplyRetention(backups, policy, now)
	require.Len(t, pruned, 1, "should prune backups strictly older than 3 days")
}

func TestApplyRetention_Combined(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	backups := makeBackups(10, now)
	// Backups at day offsets: 0, -1, -2, -3, -4, -5, -6, -7, -8, -9
	// Count-based (keep 5): prunes indices 5-9 (days -5 through -9)
	// Age-based (48h): cutoff = day -2 exactly. Prunes strictly before
	// cutoff: days -3 through -9 = indices 3-9.
	// Union: indices 3-9 = 7 pruned.
	policy := siphon.RetentionPolicy{
		KeepCount: intPtr(5),
		KeepAge:   strPtr("48h"),
	}

	pruned := siphon.ApplyRetention(backups, policy, now)
	require.Len(t, pruned, 7, "combined pruning should be the union of both rules")
}

func TestApplyRetention_EmptyBackups(t *testing.T) {
	t.Parallel()

	now := time.Now()
	pruned := siphon.ApplyRetention(nil, siphon.RetentionPolicy{KeepCount: intPtr(3)}, now)
	require.Nil(t, pruned)
}

func TestApplyRetention_NoPolicyKeepsAll(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	backups := makeBackups(5, now)

	pruned := siphon.ApplyRetention(backups, siphon.RetentionPolicy{}, now)
	require.Empty(t, pruned, "no policy means nothing is pruned")
}

func TestApplyRetention_DaySuffix(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	backups := makeBackups(10, now)
	// 3d = 72h. Cutoff = day -3 exactly. Prunes strictly before: days -4
	// through -9 = indices 4-9 = 6 backups.

	policy := siphon.RetentionPolicy{
		KeepAge: strPtr("3d"), // 3 days = 72h
	}

	pruned := siphon.ApplyRetention(backups, policy, now)
	require.Len(t, pruned, 6, "should prune backups older than 3 days")
}

func TestParseDuration_DaySuffix(t *testing.T) {
	t.Parallel()

	d, err := siphon.ParseDuration("30d")
	require.NoError(t, err)
	require.Equal(t, 30*24*time.Hour, d)
}

func TestParseDuration_StandardGo(t *testing.T) {
	t.Parallel()

	d, err := siphon.ParseDuration("720h")
	require.NoError(t, err)
	require.Equal(t, 720*time.Hour, d)
}
