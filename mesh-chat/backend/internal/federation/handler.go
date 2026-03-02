package federation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	group.GET("/server-key", h.getServerKey)
	group.GET("/.well-known", h.getWellKnown)
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

	if err := h.service.VerifyAndProcessEvent(c.Request.Context(), envelope.Event); err != nil {
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
		Processed: true,
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

func (h *Handler) getServerKey(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetServerKeyResponse())
}

func (h *Handler) getWellKnown(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetWellKnownResponse())
}
