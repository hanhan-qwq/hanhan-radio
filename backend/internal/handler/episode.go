package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hanhan-radio/backend/internal/manager"
)

// EpisodeHandler handles HTTP requests for episodes.
type EpisodeHandler struct {
	mgr *manager.Manager
	store *manager.Store
}

// New creates an EpisodeHandler.
func New(mgr *manager.Manager, store *manager.Store) *EpisodeHandler {
	return &EpisodeHandler{mgr: mgr, store: store}
}

// CreateEpisode handles POST /api/v1/episodes.
func (h *EpisodeHandler) CreateEpisode(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40000,
			"message": "prompt is required",
			"data":    nil,
		})
		return
	}

	ep, err := h.mgr.Submit(c.Request.Context(), req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50000,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "ok",
		"data":    ep,
	})
}

// GetEpisode handles GET /api/v1/episodes/:id.
func (h *EpisodeHandler) GetEpisode(c *gin.Context) {
	id := c.Param("id")

	ep, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40400,
			"message": "episode not found",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data":    ep,
	})
}

// ListEpisodes handles GET /api/v1/episodes.
func (h *EpisodeHandler) ListEpisodes(c *gin.Context) {
	eps := h.store.List()
	if eps == nil {
		eps = []*manager.Episode{}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"episodes": eps,
		},
	})
}
