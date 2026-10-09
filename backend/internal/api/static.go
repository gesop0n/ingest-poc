package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	htmlCache  = "no-cache"
	assetCache = "public, max-age=31536000, immutable"
)

func RegisterStatic(r *gin.Engine, frontend fs.FS) {
	r.NoRoute(func(c *gin.Context) {
		// APIの未定義パスはSPAへ渡さない
		if c.Request.URL.Path == "/api" ||
			strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}

		// 静的ファイルはGET/HEADのみ許可
		if c.Request.Method != http.MethodGet &&
			c.Request.Method != http.MethodHead {
			c.Status(http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(
			path.Clean(c.Request.URL.Path), "/",
		)
		if name == "" {
			name = "index.html"
		}

		// all: で埋め込んだ .gitkeep などのドットファイルは配信しない
		if strings.HasPrefix(path.Base(name), ".") {
			c.Status(http.StatusNotFound)
			return
		}

		// ファイルが存在する場合
		if info, err := fs.Stat(frontend, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				c.Header("Cache-Control", assetCache)
			} else {
				c.Header("Cache-Control", htmlCache)
			}

			http.ServeFileFS(c.Writer, c.Request, frontend, name)
			return
		}

		// 存在しないアセットやファイルは404
		if strings.HasPrefix(name, "assets/") ||
			path.Ext(name) != "" {
			c.Status(http.StatusNotFound)
			return
		}

		// HTML画面遷移以外はフォールバックしない
		if !strings.Contains(
			c.GetHeader("Accept"), "text/html",
		) {
			c.Status(http.StatusNotFound)
			return
		}

		// React Routerへ処理を委譲
		c.Header("Cache-Control", htmlCache)
		http.ServeFileFS(c.Writer, c.Request, frontend, "index.html")
	})
}
