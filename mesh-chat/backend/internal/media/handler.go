package media

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meshchat/backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/media/upload", h.upload)
	router.GET("/media/*objectPath", h.redirect)
}

func (h *Handler) upload(c *gin.Context) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "missing file field"})
		return
	}

	response, err := h.service.Upload(c.Request.Context(), userID, fileHeader)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *Handler) redirect(c *gin.Context) {
	objectPath := strings.TrimPrefix(c.Param("objectPath"), "/")
	if objectPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_object_path"})
		return
	}

	url, err := h.service.PresignedURL(objectPath, 15*time.Minute)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrMediaUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "media_unavailable", "message": err.Error()})
	case errors.Is(err, ErrInvalidMimeType):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "invalid_media_type", "message": err.Error()})
	case errors.Is(err, ErrFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file_too_large", "message": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "media_error", "message": err.Error()})
	}
}
