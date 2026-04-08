package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandTableGroup_SimpleList(t *testing.T) {
	t.Parallel()
	group := &TableGroup{
		Tables: []string{"users", "orders", "products"},
	}
	allTables := []string{"users", "orders", "products", "logs", "sessions"}

	result, err := ExpandTableGroup(group, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"users", "orders", "products"}, result)
}

func TestExpandTableGroup_SimpleListWithExclude(t *testing.T) {
	t.Parallel()
	group := &TableGroup{
		Tables:  []string{"users", "orders", "products"},
		Exclude: []string{"orders"},
	}
	allTables := []string{"users", "orders", "products", "logs"}

	result, err := ExpandTableGroup(group, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"users", "products"}, result)
}

func TestExpandTableGroup_RegexMatch(t *testing.T) {
	t.Parallel()
	group := &TableGroup{
		Match: strPtr("^log_"),
	}
	allTables := []string{"users", "orders", "log_access", "log_error", "log_audit"}

	result, err := ExpandTableGroup(group, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"log_access", "log_error", "log_audit"}, result)
}

func TestExpandTableGroup_RegexWithExclude(t *testing.T) {
	t.Parallel()
	group := &TableGroup{
		Match: strPtr("^log_"),
		Exclude: []string{"log_audit"},
	}
	allTables := []string{"users", "log_access", "log_error", "log_audit"}

	result, err := ExpandTableGroup(group, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"log_access", "log_error"}, result)
}

func TestExpandTableGroup_InvalidRegex(t *testing.T) {
	t.Parallel()
	group := &TableGroup{
		Match: strPtr("[invalid"),
	}
	allTables := []string{"users"}

	_, err := ExpandTableGroup(group, allTables)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid table group pattern")
}

func TestExpandTableGroup_Nil(t *testing.T) {
	t.Parallel()
	result, err := ExpandTableGroup(nil, []string{"users"})
	require.NoError(t, err)
	require.Nil(t, result)
}

// --- ResolveTableSelection tests ---

func TestResolveTableSelection_AllTablesOverride(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "logs"}
	opts := TableSelectionOptions{
		AllTables: true,
		Tables:    []string{"users"}, // should be ignored
	}

	result, err := ResolveTableSelection(opts, nil, allTables)
	require.NoError(t, err)
	require.Equal(t, allTables, result)
}

func TestResolveTableSelection_GroupsWithExclude(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "products", "log_access", "log_error"}
	groups := map[string]*TableGroup{
		"core": {Tables: []string{"users", "orders", "products"}},
		"logs": {Match: strPtr("^log_")},
	}
	opts := TableSelectionOptions{
		Groups:        []string{"core"},
		ExcludeGroups: []string{"logs"},
	}

	result, err := ResolveTableSelection(opts, groups, allTables)
	require.NoError(t, err)
	// core group minus logs (logs don't overlap with core, so all core tables remain).
	require.Equal(t, []string{"orders", "products", "users"}, result)
}

func TestResolveTableSelection_GroupsUnionWithTables(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "products", "settings"}
	groups := map[string]*TableGroup{
		"core": {Tables: []string{"users", "orders"}},
	}
	opts := TableSelectionOptions{
		Groups: []string{"core"},
		Tables: []string{"settings"}, // union with groups
	}

	result, err := ResolveTableSelection(opts, groups, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"orders", "settings", "users"}, result)
}

func TestResolveTableSelection_ExcludeOnlyStartsFromAll(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "log_access", "log_error"}
	groups := map[string]*TableGroup{
		"logs": {Match: strPtr("^log_")},
	}
	opts := TableSelectionOptions{
		ExcludeGroups: []string{"logs"},
	}

	result, err := ResolveTableSelection(opts, groups, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"orders", "users"}, result)
}

func TestResolveTableSelection_TablesAlone(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "products"}
	opts := TableSelectionOptions{
		Tables: []string{"users", "orders"},
	}

	result, err := ResolveTableSelection(opts, nil, allTables)
	require.NoError(t, err)
	require.Equal(t, []string{"users", "orders"}, result)
}

func TestResolveTableSelection_DefaultAll(t *testing.T) {
	t.Parallel()
	allTables := []string{"users", "orders", "products"}
	opts := TableSelectionOptions{}

	result, err := ResolveTableSelection(opts, nil, allTables)
	require.NoError(t, err)
	require.Equal(t, allTables, result)
}

func TestResolveTableSelection_MissingGroup(t *testing.T) {
	t.Parallel()
	allTables := []string{"users"}
	opts := TableSelectionOptions{
		Groups: []string{"nonexistent"},
	}

	_, err := ResolveTableSelection(opts, nil, allTables)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestResolveTableSelection_MissingExcludeGroup(t *testing.T) {
	t.Parallel()
	allTables := []string{"users"}
	opts := TableSelectionOptions{
		ExcludeGroups: []string{"nonexistent"},
	}

	_, err := ResolveTableSelection(opts, nil, allTables)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
