package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopoSort_LinearChain(t *testing.T) {
	t.Parallel()
	// C depends on B, B depends on A => A, B, C
	deps := []TableDependency{
		{Table: "C", DependsOn: []string{"B"}},
		{Table: "B", DependsOn: []string{"A"}},
		{Table: "A", DependsOn: nil},
	}

	result, err := TopoSort(deps)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// A must come before B, B must come before C.
	indexOf := make(map[string]int)
	for i, t := range result {
		indexOf[t] = i
	}
	require.Less(t, indexOf["A"], indexOf["B"])
	require.Less(t, indexOf["B"], indexOf["C"])
}

func TestTopoSort_DiamondDependency(t *testing.T) {
	t.Parallel()
	// D depends on B and C; B and C both depend on A.
	deps := []TableDependency{
		{Table: "A", DependsOn: nil},
		{Table: "B", DependsOn: []string{"A"}},
		{Table: "C", DependsOn: []string{"A"}},
		{Table: "D", DependsOn: []string{"B", "C"}},
	}

	result, err := TopoSort(deps)
	require.NoError(t, err)
	require.Len(t, result, 4)

	indexOf := make(map[string]int)
	for i, t := range result {
		indexOf[t] = i
	}
	require.Less(t, indexOf["A"], indexOf["B"])
	require.Less(t, indexOf["A"], indexOf["C"])
	require.Less(t, indexOf["B"], indexOf["D"])
	require.Less(t, indexOf["C"], indexOf["D"])
}

func TestTopoSort_NoDependencies(t *testing.T) {
	t.Parallel()
	deps := []TableDependency{
		{Table: "users", DependsOn: nil},
		{Table: "products", DependsOn: nil},
		{Table: "settings", DependsOn: nil},
	}

	result, err := TopoSort(deps)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// All tables should be present regardless of order.
	tableSet := make(map[string]bool)
	for _, t := range result {
		tableSet[t] = true
	}
	require.True(t, tableSet["users"])
	require.True(t, tableSet["products"])
	require.True(t, tableSet["settings"])
}

func TestTopoSort_CycleDetection(t *testing.T) {
	t.Parallel()
	deps := []TableDependency{
		{Table: "A", DependsOn: []string{"B"}},
		{Table: "B", DependsOn: []string{"C"}},
		{Table: "C", DependsOn: []string{"A"}},
	}

	_, err := TopoSort(deps)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTransferFailed)
	require.Contains(t, err.Error(), "cycle")
}

func TestTopoSort_Empty(t *testing.T) {
	t.Parallel()
	result, err := TopoSort(nil)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestTopoSort_SingleTable(t *testing.T) {
	t.Parallel()
	deps := []TableDependency{
		{Table: "users", DependsOn: nil},
	}

	result, err := TopoSort(deps)
	require.NoError(t, err)
	require.Equal(t, []string{"users"}, result)
}
