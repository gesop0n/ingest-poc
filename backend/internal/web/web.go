package web

import (
	"embed"
	"fmt"
	"io/fs"
)

// dist はビルド時に frontend/dist からコピーされる
// .gitkeep だけの状態でもコンパイルできるよう all: を付ける
//
//go:embed all:dist
var assets embed.FS

func FS() (fs.FS, error) {
	frontendFS, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, fmt.Errorf("initialize frontend filesystem: %w", err)
	}

	if _, err := fs.Stat(frontendFS, "index.html"); err != nil {
		return nil, fmt.Errorf("frontend index.html (run `make build`): %w", err)
	}

	return frontendFS, nil
}
