package api

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(frontend fs.FS) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok:",
			})
		})
	}

	if frontend != nil {
		RegisterStatic(r, frontend)
	}

	return r
}
