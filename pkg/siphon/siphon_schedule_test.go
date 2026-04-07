package siphon_test

import (
	"context"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func newScheduleTestSiphon(t *testing.T) (*siphon.Siphon, *sandbox.Sandbox) {
	t.Helper()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, sandbox.WithFixture("testuser", "~"))

	s, err := siphon.New(siphon.SiphonOptions{
		Runtime: sb.Runtime(),
	})
	require.NoError(t, err)
	return s, sb
}

func TestCreateSchedule_Success(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "nightly-dev",
		Connection: "dev",
		Repo:       "nightly",
		Time:       "02:00",
		Interval:   "daily",
		Backend:    "launchd",
	})
	require.NoError(t, err)

	// Verify it appears in list.
	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.Equal(t, "nightly-dev", schedules[0].Name)
	require.Equal(t, "dev", schedules[0].Connection)
	require.Equal(t, "nightly", schedules[0].Repo)
	require.Equal(t, "02:00", schedules[0].Time)
	require.Equal(t, "daily", schedules[0].Interval)
	require.Equal(t, "launchd", schedules[0].Backend)
	require.True(t, schedules[0].Active, "should be active because plist was written")
}

func TestCreateSchedule_CronBackend(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "hourly-dev",
		Connection: "dev",
		Time:       "00:15",
		Interval:   "hourly",
		Backend:    "cron",
	})
	require.NoError(t, err)

	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.Equal(t, "cron", schedules[0].Backend)
	require.True(t, schedules[0].Active, "cron schedules are always reported active")
}

func TestCreateSchedule_Defaults(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	// Minimal options: name and connection only.
	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "minimal",
		Connection: "dev",
	})
	require.NoError(t, err)

	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.Equal(t, "02:00", schedules[0].Time, "default time")
	require.Equal(t, "daily", schedules[0].Interval, "default interval")
	require.Equal(t, "launchd", schedules[0].Backend, "default backend")
}

func TestCreateSchedule_WithRetention(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	keepCount := 5
	keepAge := "30d"
	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "retained",
		Connection: "dev",
		Retention: &siphon.RetentionPolicy{
			KeepCount: &keepCount,
			KeepAge:   &keepAge,
		},
	})
	require.NoError(t, err)

	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.NotNil(t, schedules[0].Retention)
	require.Equal(t, 5, *schedules[0].Retention.KeepCount)
	require.Equal(t, "30d", *schedules[0].Retention.KeepAge)
}

func TestCreateSchedule_DuplicateName(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "dup",
		Connection: "dev",
	})
	require.NoError(t, err)

	err = s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "dup",
		Connection: "dev",
	})
	require.ErrorIs(t, err, siphon.ErrScheduleExists)
}

func TestCreateSchedule_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "bad-conn",
		Connection: "nonexistent",
	})
	require.ErrorIs(t, err, siphon.ErrConnectionNotFound)
}

func TestCreateSchedule_RepoNotFound(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "bad-repo",
		Connection: "dev",
		Repo:       "nonexistent",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not configured")
}

func TestCreateSchedule_InvalidTime(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "bad-time",
		Connection: "dev",
		Time:       "25:99",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid time")
}

func TestRemoveSchedule_Success(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	// Create first.
	err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
		Name:       "to-remove",
		Connection: "dev",
	})
	require.NoError(t, err)

	// Remove.
	err = s.RemoveSchedule(ctx, &siphon.RemoveScheduleOptions{
		Name: "to-remove",
	})
	require.NoError(t, err)

	// Verify gone.
	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Empty(t, schedules)
}

func TestRemoveSchedule_NotFound(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	err := s.RemoveSchedule(ctx, &siphon.RemoveScheduleOptions{
		Name: "nonexistent",
	})
	require.ErrorIs(t, err, siphon.ErrScheduleNotFound)
}

func TestListSchedules_Empty(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Empty(t, schedules)
}

func TestListSchedules_Sorted(t *testing.T) {
	t.Parallel()
	s, _ := newScheduleTestSiphon(t)
	ctx := context.Background()

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		err := s.CreateSchedule(ctx, &siphon.CreateScheduleOptions{
			Name:       name,
			Connection: "dev",
		})
		require.NoError(t, err)
	}

	schedules, err := s.ListSchedules(ctx, &siphon.ListSchedulesOptions{})
	require.NoError(t, err)
	require.Len(t, schedules, 3)
	require.Equal(t, "alpha", schedules[0].Name)
	require.Equal(t, "bravo", schedules[1].Name)
	require.Equal(t, "charlie", schedules[2].Name)
}
