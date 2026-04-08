package siphon

import (
	"fmt"
	"regexp"
	"sort"
)

// TableSelectionOptions configures how tables are selected for transfer.
type TableSelectionOptions struct {
	AllTables     bool     // override: transfer all tables
	Tables        []string // explicit table names
	Groups        []string // named table groups to include
	ExcludeGroups []string // named table groups to exclude
}

// ResolveTableSelection implements the four-level precedence:
//  1. --all-tables overrides everything: return allTables
//  2. --group + --exclude-group + --tables union
//  3. --tables alone
//  4. default: all tables
func ResolveTableSelection(opts TableSelectionOptions, groups map[string]*TableGroup, allTables []string) ([]string, error) {
	// Level 1: --all-tables overrides everything.
	if opts.AllTables {
		return allTables, nil
	}

	hasGroups := len(opts.Groups) > 0
	hasExclude := len(opts.ExcludeGroups) > 0
	hasTables := len(opts.Tables) > 0

	// Level 2: groups + excludes + explicit tables union.
	if hasGroups || hasExclude {
		included := make(map[string]bool)

		// Expand included groups.
		for _, groupName := range opts.Groups {
			group, ok := groups[groupName]
			if !ok {
				return nil, fmt.Errorf("table group %q not found", groupName)
			}
			expanded, err := ExpandTableGroup(group, allTables)
			if err != nil {
				return nil, fmt.Errorf("expanding group %q: %w", groupName, err)
			}
			for _, t := range expanded {
				included[t] = true
			}
		}

		// If no groups specified but excludes are, start with all tables.
		if !hasGroups && hasExclude {
			for _, t := range allTables {
				included[t] = true
			}
		}

		// Add explicit tables to the union.
		for _, t := range opts.Tables {
			included[t] = true
		}

		// Remove excluded groups.
		for _, groupName := range opts.ExcludeGroups {
			group, ok := groups[groupName]
			if !ok {
				return nil, fmt.Errorf("exclude group %q not found", groupName)
			}
			expanded, err := ExpandTableGroup(group, allTables)
			if err != nil {
				return nil, fmt.Errorf("expanding exclude group %q: %w", groupName, err)
			}
			for _, t := range expanded {
				delete(included, t)
			}
		}

		return sortedKeys(included), nil
	}

	// Level 3: explicit tables alone.
	if hasTables {
		return opts.Tables, nil
	}

	// Level 4: default is all tables.
	return allTables, nil
}

// ExpandTableGroup expands a TableGroup against the full table list.
// Simple format: literal table name list.
// Regex format: match pattern against all table names.
func ExpandTableGroup(group *TableGroup, allTables []string) ([]string, error) {
	if group == nil {
		return nil, nil
	}

	var result []string

	if group.Match != nil && *group.Match != "" {
		// Regex mode: match pattern against all table names.
		re, err := regexp.Compile(*group.Match)
		if err != nil {
			return nil, fmt.Errorf("invalid table group pattern %q: %w", *group.Match, err)
		}

		excludeSet := make(map[string]bool)
		for _, e := range group.Exclude {
			excludeSet[e] = true
		}

		for _, t := range allTables {
			if re.MatchString(t) && !excludeSet[t] {
				result = append(result, t)
			}
		}
	} else {
		// Simple mode: literal table name list.
		excludeSet := make(map[string]bool)
		for _, e := range group.Exclude {
			excludeSet[e] = true
		}

		for _, t := range group.Tables {
			if !excludeSet[t] {
				result = append(result, t)
			}
		}
	}

	return result, nil
}

// sortedKeys returns sorted keys from a boolean map.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
