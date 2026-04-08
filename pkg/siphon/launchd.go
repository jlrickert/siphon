package siphon

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
)

// LaunchdConfig holds the parameters needed to generate a macOS launchd plist.
type LaunchdConfig struct {
	Label                 string         // e.g., com.siphon.backup.<name>
	Program               string         // path to siphon binary
	Arguments             []string       // e.g., backup create <connection> --repo <repo>
	StartCalendarInterval map[string]int // Hour, Minute, Weekday, etc.
	StandardOutPath       string         // log file for stdout
	StandardErrorPath     string         // log file for stderr
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Program}}</string>
{{- range .Arguments}}
		<string>{{.}}</string>
{{- end}}
	</array>
	<key>StartCalendarInterval</key>
	<dict>
{{- range $key, $val := .StartCalendarInterval}}
		<key>{{$key}}</key>
		<integer>{{$val}}</integer>
{{- end}}
	</dict>
{{- if .StandardOutPath}}
	<key>StandardOutPath</key>
	<string>{{.StandardOutPath}}</string>
{{- end}}
{{- if .StandardErrorPath}}
	<key>StandardErrorPath</key>
	<string>{{.StandardErrorPath}}</string>
{{- end}}
</dict>
</plist>
`

// GeneratePlist renders a launchd plist XML document from the given config.
func GeneratePlist(cfg LaunchdConfig) ([]byte, error) {
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing plist template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("executing plist template: %w", err)
	}

	return buf.Bytes(), nil
}

// LaunchAgentsDir returns the standard macOS LaunchAgents directory for the
// given home directory.
func LaunchAgentsDir(home string) string {
	return filepath.Join(home, "Library", "LaunchAgents")
}

// PlistFilename returns the conventional plist filename for a schedule name.
func PlistFilename(name string) string {
	return fmt.Sprintf("com.siphon.backup.%s.plist", name)
}

// PlistLabel returns the conventional launchd label for a schedule name.
func PlistLabel(name string) string {
	return fmt.Sprintf("com.siphon.backup.%s", name)
}
