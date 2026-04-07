package cli

var (
	// Version is set at build time via -ldflags or defaults to "dev".
	Version = "dev"

	// LicenseText is populated by the main package from an embedded LICENSE file.
	LicenseText string
)
