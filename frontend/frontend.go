package frontend

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed dist
var assets embed.FS

func FS() (fs.FS, error) {
	frontendFS, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, fmt.Errorf("initialize frontend filesystem: %w", err)
	}

	if _, err := fs.Stat(frontendFS, "index.html"); err != nil {
		return nil, fmt.Errorf("frontend index.html: %w", err)
	}

	return frontendFS, nil
}
