package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePolicy_DefaultAllow(t *testing.T) {
	t.Parallel()
	action := ResolvePolicy(nil, "prod", SurfaceCLI)
	require.Equal(t, PolicyAllow, action)
}

func TestResolvePolicy_EmptyPolicies(t *testing.T) {
	t.Parallel()
	policies := map[string]*PolicyConfig{}
	action := ResolvePolicy(policies, "prod", SurfaceMCP)
	require.Equal(t, PolicyAllow, action)
}

func TestResolvePolicy_PerConnectionPerSurface(t *testing.T) {
	t.Parallel()
	deny := PolicyDeny
	policies := map[string]*PolicyConfig{
		"prod": {
			MCP: &deny,
		},
	}

	// MCP surface should be denied.
	action := ResolvePolicy(policies, "prod", SurfaceMCP)
	require.Equal(t, PolicyDeny, action)

	// CLI surface not set -> falls through to default (allow).
	action = ResolvePolicy(policies, "prod", SurfaceCLI)
	require.Equal(t, PolicyAllow, action)
}

func TestResolvePolicy_GlobalDefault(t *testing.T) {
	t.Parallel()
	confirm := PolicyConfirm
	policies := map[string]*PolicyConfig{
		"__default__": {
			MCP: &confirm,
		},
	}

	// Unmatched connection falls through to global default.
	action := ResolvePolicy(policies, "staging", SurfaceMCP)
	require.Equal(t, PolicyConfirm, action)

	// API surface not in global default -> hardcoded allow.
	action = ResolvePolicy(policies, "staging", SurfaceAPI)
	require.Equal(t, PolicyAllow, action)
}

func TestResolvePolicy_ConnectionOverridesGlobal(t *testing.T) {
	t.Parallel()
	deny := PolicyDeny
	confirm := PolicyConfirm
	policies := map[string]*PolicyConfig{
		"__default__": {
			CLI: &confirm,
		},
		"prod": {
			CLI: &deny,
		},
	}

	// Per-connection override takes priority.
	action := ResolvePolicy(policies, "prod", SurfaceCLI)
	require.Equal(t, PolicyDeny, action)

	// Other connections use global default.
	action = ResolvePolicy(policies, "dev", SurfaceCLI)
	require.Equal(t, PolicyConfirm, action)
}

func TestResolvePolicy_ReadonlyAction(t *testing.T) {
	t.Parallel()
	readonly := PolicyReadonly
	policies := map[string]*PolicyConfig{
		"analytics": {
			API: &readonly,
		},
	}

	action := ResolvePolicy(policies, "analytics", SurfaceAPI)
	require.Equal(t, PolicyReadonly, action)
}

func TestResolvePolicy_AllSurfaces(t *testing.T) {
	t.Parallel()
	deny := PolicyDeny
	confirm := PolicyConfirm
	allow := PolicyAllow

	policies := map[string]*PolicyConfig{
		"prod": {
			CLI: &allow,
			MCP: &confirm,
			API: &deny,
		},
	}

	require.Equal(t, PolicyAllow, ResolvePolicy(policies, "prod", SurfaceCLI))
	require.Equal(t, PolicyConfirm, ResolvePolicy(policies, "prod", SurfaceMCP))
	require.Equal(t, PolicyDeny, ResolvePolicy(policies, "prod", SurfaceAPI))
}
