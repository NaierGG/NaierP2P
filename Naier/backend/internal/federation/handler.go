package federation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/naier/backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/_federation/v1")
	group.POST("/events", h.receiveEvent)
	group.GET("/users/:username", h.getUser)
	group.GET("/channels/:id/state", h.getChannelState)
	group.GET("/server-key", h.getServerKey)
	group.GET("/.well-known", h.getWellKnown)
}

func (h *Handler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	group := router.Group("/federation")
	group.POST("/channels/:id/sync", h.pushChannelState)
	group.POST("/channels/:id/pull", h.pullChannelState)
}

func (h *Handler) receiveEvent(c *gin.Context) {
	var envelope EventEnvelope
	if err := c.ShouldBindJSON(&envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	result, err := h.service.VerifyAndProcessEvent(c.Request.Context(), envelope.Event)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_event",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, EventAckResponse{
		Status:    "accepted",
		EventID:   envelope.Event.EventID,
		ServerID:  envelope.Event.ServerID,
		Processed: result.Processed,
		Duplicate: result.Duplicate,
	})
}

func (h *Handler) getUser(c *gin.Context) {
	user, err := h.service.GetLocalUser(c.Request.Context(), c.Param("username"))
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "user_not_found",
			"message": "user not found",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "user_lookup_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, RemoteUserResponse{User: *user})
}

func (h *Handler) getChannelState(c *gin.Context) {
	state, err := h.service.GetLocalChannelState(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "channel_state_not_found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ChannelStateResponse{Channel: state})
}

func (h *Handler) pushChannelState(c *gin.Context) {
	actorID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_channel_id", "message": "invalid channel id"})
		return
	}

	var req ChannelStateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if err := h.service.SendChannelStateToServer(c.Request.Context(), actorID, channelID, req.TargetDomain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_state_sync_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted", "target_domain": req.TargetDomain})
}

func (h *Handler) pullChannelState(c *gin.Context) {
	actorID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_channel_id", "message": "invalid channel id"})
		return
	}

	var req ChannelStateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	state, err := h.service.PullChannelStateFromServer(c.Request.Context(), actorID, channelID, req.TargetDomain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_state_pull_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ChannelStateResponse{Channel: state})
}

func (h *Handler) getServerKey(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetServerKeyResponse())
}

func (h *Handler) getWellKnown(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetWellKnownResponse())
}
