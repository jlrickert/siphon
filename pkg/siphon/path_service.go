package siphon

import (
	"path/filepath"

	"github.com/jlrickert/cli-toolkit/appctx"
	"github.com/jlrickert/cli-toolkit/toolkit"
)

// PathService provides standard path resolution for siphon config and state
// directories.
type PathService struct {
	*appctx.AppPaths
}

// NewPathService creates a PathService rooted at root.
func NewPathService(rt *toolkit.Runtime, root string) (*PathService, error) {
	appPaths, err := appctx.NewAppPaths(rt, root, DefaultAppName)
	if err != nil {
		return nil, err
	}
	return &PathService{AppPaths: appPaths}, nil
}

// Project returns the project-local config root directory.
func (s *PathService) Project() string {
	return s.AppPaths.LocalConfigRoot
}

// ProjectConfig returns the path to the project-level config file.
func (s *PathService) ProjectConfig() string {
	return filepath.Join(s.LocalConfigRoot, "config.yaml")
}

// UserConfig returns the path to the user-level config file.
func (s *PathService) UserConfig() string {
	return filepath.Join(s.ConfigRoot, "config.yaml")
}
