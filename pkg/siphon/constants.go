package siphon

const (
	// DefaultAppName is the base directory name used for siphon user
	// configuration and state directories (e.g., ~/.config/siphon,
	// ~/.local/state/siphon).
	DefaultAppName = "siphon"

	// siphonEnvPrefix is the environment variable prefix for config overrides.
	siphonEnvPrefix = "SIPHON_"
)

// siphonEnvVarKeys lists the environment variables consulted during config
// cascade resolution.
var siphonEnvVarKeys = []string{
	"SIPHON_DEFAULT_CONNECTION",
	"SIPHON_LOG_FILE",
	"SIPHON_LOG_LEVEL",
}
