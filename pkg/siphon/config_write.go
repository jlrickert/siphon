package siphon

import (
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"gopkg.in/yaml.v3"
)

// WriteConfig writes a Config struct to a YAML file at path. The Runtime's
// WriteFile method creates parent directories as needed.
func WriteConfig(rt *toolkit.Runtime, path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := rt.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config to %s: %w", path, err)
	}
	return nil
}
