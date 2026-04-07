package siphon_test

import (
	"testing"
	"time"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestResolveBackupName_DefaultTemplate(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 4, 7, 14, 30, 15, 0, time.UTC)
	data := siphon.NewBackupNameData("dev", "testdb", "mariadb", "logical", ts)

	name, err := siphon.ResolveBackupName("", data)
	require.NoError(t, err)
	require.Equal(t, "dev-2026-04-07-14-30-15", name)
}

func TestResolveBackupName_CustomTemplate(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 4, 7, 14, 30, 15, 0, time.UTC)
	data := siphon.NewBackupNameData("prod", "mydb", "postgresql", "logical", ts)

	name, err := siphon.ResolveBackupName("{{.Engine}}-{{.Database}}-{{.Date}}", data)
	require.NoError(t, err)
	require.Equal(t, "postgresql-mydb-2026-04-07", name)
}

func TestResolveBackupName_AllFields(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 9, 5, 0, 0, time.UTC)
	data := siphon.NewBackupNameData("staging", "appdb", "mariadb", "physical", ts)

	name, err := siphon.ResolveBackupName("{{.Connection}}-{{.Engine}}-{{.Type}}-{{.Date}}-{{.Time}}", data)
	require.NoError(t, err)
	require.Equal(t, "staging-mariadb-physical-2026-01-15-09-05-00", name)
}

func TestResolveBackupName_InvalidTemplate(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	data := siphon.NewBackupNameData("dev", "testdb", "mariadb", "logical", ts)

	_, err := siphon.ResolveBackupName("{{.Invalid", data)
	require.Error(t, err)
}

func TestNewBackupNameData(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 12, 25, 23, 59, 59, 0, time.UTC)
	data := siphon.NewBackupNameData("prod", "maindb", "postgresql", "logical", ts)

	require.Equal(t, "prod", data.Connection)
	require.Equal(t, "maindb", data.Database)
	require.Equal(t, "postgresql", data.Engine)
	require.Equal(t, "logical", data.Type)
	require.Equal(t, "2026-12-25", data.Date)
	require.Equal(t, "23-59-59", data.Time)
	require.True(t, data.Timestamp.Equal(ts))
}
