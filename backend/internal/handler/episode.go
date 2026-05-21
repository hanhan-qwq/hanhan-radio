package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"hanhan-radio/backend/internal/manager"
)

// EpisodeHandler handles HTTP requests for episodes.
type EpisodeHandler struct {
	mgr   *manager.Manager
	store *manager.Store
}

// New creates an EpisodeHandler.
func New(mgr *manager.Manager, store *manager.Store) *EpisodeHandler {
	return &EpisodeHandler{mgr: mgr, store: store}
}

// CreateEpisode godoc
// @Summary      创建 episode
// @Description  提交用户输入文本，后台异步生成电台音频，立即返回 episode ID
// @Tags         episodes
// @Accept       json
// @Produce      json
// @Param        body  body      CreateEpisodeRequest  true  "用户输入"
// @Success      201   {object}  Envelope{data=manager.Episode}  "创建成功"
// @Failure      400   {object}  ErrorEnvelope           "参数错误"
// @Router       /episodes [post]
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

// GetEpisode godoc
// @Summary      查询单个 episode
// @Description  根据 ID 查询 episode 状态和结果。前端轮询此接口直到 status 为 done 或 failed。
// @Tags         episodes
// @Produce      json
// @Param        id   path      string  true  "episode ID"
// @Success      200  {object}  Envelope{data=manager.Episode}  "成功"
// @Failure      404  {object}  ErrorEnvelope           "episode 不存在"
// @Router       /episodes/{id} [get]
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

// ListEpisodes godoc
// @Summary      获取 episode 列表
// @Description  按创建时间倒序返回最近 episode 列表（排除失败的），最多 50 条。列表中的 segments 不含 segue 全文。
// @Tags         episodes
// @Produce      json
// @Success      200  {object}  Envelope{data=EpisodeList}  "成功"
// @Router       /episodes [get]
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

// ---- swagger types (not real handlers, only for document generation) ----

// CreateEpisodeRequest is the request body for POST /episodes.
type CreateEpisodeRequest struct {
	Prompt string `json:"prompt" example:"想听点轻松的"`
}

// Envelope is the unified API response wrapper.
type Envelope struct {
	Code    int         `json:"code"    example:"0"`
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data"`
}

// ErrorEnvelope is the unified error response.
type ErrorEnvelope struct {
	Code    int    `json:"code"    example:"40000"`
	Message string `json:"message" example:"prompt is required"`
	Data    *struct{} `json:"data"`
}

// EpisodeList wraps the episode list response.
type EpisodeList struct {
	Episodes []manager.Episode `json:"episodes"`
}
