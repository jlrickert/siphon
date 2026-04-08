package siphon

import "fmt"

// TableDependency describes the foreign key dependencies for a single table.
type TableDependency struct {
	Table     string   // table name
	DependsOn []string // tables this table has foreign keys to
}

// TopoSort returns tables in dependency order (dependencies first).
// It returns an error if a cycle is detected.
func TopoSort(deps []TableDependency) ([]string, error) {
	// Build adjacency list and in-degree map.
	graph := make(map[string][]string) // table -> tables that depend on it
	inDegree := make(map[string]int)   // table -> number of dependencies
	allTables := make(map[string]bool)

	for _, d := range deps {
		allTables[d.Table] = true
		for _, dep := range d.DependsOn {
			allTables[dep] = true
		}
	}

	// Initialize all tables with zero in-degree.
	for t := range allTables {
		if _, ok := inDegree[t]; !ok {
			inDegree[t] = 0
		}
	}

	// Build the graph: if table A depends on table B, then B -> A in
	// the graph, and A's in-degree increments.
	for _, d := range deps {
		for _, dep := range d.DependsOn {
			graph[dep] = append(graph[dep], d.Table)
			inDegree[d.Table]++
		}
	}

	// Kahn's algorithm for topological sort.
	var queue []string
	for t, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, t)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(allTables) {
		return nil, fmt.Errorf("%w: cycle detected in table dependencies", ErrTransferFailed)
	}

	return result, nil
}
