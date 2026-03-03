package message

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naier/backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/events/sync", h.syncEvents)
	router.GET("/channels/:id/messages", h.list)
	router.GET("/channels/:id/sync", h.syncChannel)
	router.POST("/channels/:id/messages", h.create)
	router.PUT("/messages/:id", h.update)
	router.DELETE("/messages/:id", h.delete)
	router.POST("/messages/:id/reactions", h.addReaction)
	router.DELETE("/messages/:id/reactions/:emoji", h.removeReaction)
}

func (h *Handler) syncEvents(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	deviceID, err := auth.DeviceIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 100
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}

	response, err := h.service.SyncEvents(c.Request.Context(), userID, deviceID, c.Query("after"), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) list(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_channel_id"})
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}

	response, err := h.service.ListByChannel(c.Request.Context(), channelID, userID, c.Query("cursor"), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) syncChannel(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	deviceID, err := auth.DeviceIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_channel_id"})
		return
	}

	limit := 100
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}

	response, err := h.service.SyncChannelEvents(c.Request.Context(), userID, deviceID, channelID, c.Query("after"), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) create(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_channel_id"})
		return
	}

	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	message, err := h.service.CreateHTTP(c.Request.Context(), channelID, userID, req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *Handler) update(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_message_id"})
		return
	}

	var req UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	message, err := h.service.UpdateHTTP(c.Request.Context(), messageID, userID, req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, message)
}

func (h *Handler) delete(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_message_id"})
		return
	}

	message, err := h.service.DeleteHTTP(c.Request.Context(), messageID, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, message)
}

func (h *Handler) addReaction(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_message_id"})
		return
	}

	var req ReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if _, _, err := h.service.AddReaction(c.Request.Context(), userID, messageID, req.Emoji); err != nil {
		h.respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) removeReaction(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_message_id"})
		return
	}

	if _, _, err := h.service.RemoveReaction(c.Request.Context(), userID, messageID, c.Param("emoji")); err != nil {
		h.respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrMessageNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "message_not_found", "message": err.Error()})
	case errors.Is(err, ErrMessageDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_error", "message": err.Error()})
	}
}
