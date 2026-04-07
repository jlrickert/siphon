package siphon

// PolicyAction represents the action to take for a policy check.
type PolicyAction string

const (
	// PolicyAllow permits the operation without restriction.
	PolicyAllow PolicyAction = "allow"

	// PolicyConfirm requires explicit user confirmation before proceeding.
	PolicyConfirm PolicyAction = "confirm"

	// PolicyDeny blocks the operation entirely.
	PolicyDeny PolicyAction = "deny"

	// PolicyReadonly permits only read operations.
	PolicyReadonly PolicyAction = "readonly"
)

// Surface identifies the interface surface through which an operation is invoked.
type Surface string

const (
	SurfaceCLI Surface = "cli"
	SurfaceMCP Surface = "mcp"
	SurfaceAPI Surface = "api"
)

// ResolvePolicy determines the effective policy action for a connection on a
// given surface. Resolution cascade:
//
//  1. Per-connection, per-surface override (policies[connection].{surface})
//  2. Global default ("__default__" policy entry) per-surface
//  3. Hardcoded default: PolicyAllow
func ResolvePolicy(policies map[string]*PolicyConfig, connection string, surface Surface) PolicyAction {
	// 1. Check per-connection, per-surface override.
	if connPolicy, ok := policies[connection]; ok && connPolicy != nil {
		if action := surfaceAction(connPolicy, surface); action != nil {
			return *action
		}
	}

	// 2. Check global default per-surface override.
	if globalPolicy, ok := policies["__default__"]; ok && globalPolicy != nil {
		if action := surfaceAction(globalPolicy, surface); action != nil {
			return *action
		}
	}

	// 3. Hardcoded default.
	return PolicyAllow
}

// surfaceAction returns the policy action for a specific surface, or nil if
// not configured.
func surfaceAction(p *PolicyConfig, surface Surface) *PolicyAction {
	switch surface {
	case SurfaceCLI:
		return p.CLI
	case SurfaceMCP:
		return p.MCP
	case SurfaceAPI:
		return p.API
	default:
		return nil
	}
}
