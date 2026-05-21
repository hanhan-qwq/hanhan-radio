package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"hanhan-radio/backend/internal/handler"
)

// New creates a gin Engine with all routes registered.
func New(h *handler.EpisodeHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	api := r.Group("/api/v1")
	{
		api.POST("/episodes", h.CreateEpisode)
		api.GET("/episodes", h.ListEpisodes)
		api.GET("/episodes/:id", h.GetEpisode)
	}

	r.Static("/static", "./output")

	return r
}
