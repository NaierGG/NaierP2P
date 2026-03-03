package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	router.POST("/challenge", h.challenge)
	router.POST("/register", h.register)
	router.POST("/login", h.login)
	router.POST("/refresh", h.refresh)
	router.POST("/logout", authMiddleware, h.logout)
	router.GET("/me", authMiddleware, h.me)
	router.PUT("/me", authMiddleware, h.updateProfile)
	router.GET("/devices", authMiddleware, h.devices)
	router.POST("/devices/pending", authMiddleware, h.registerPendingDevice)
	router.POST("/devices/approve", authMiddleware, h.approveDevice)
	router.DELETE("/devices/:id", authMiddleware, h.revokeDevice)
}

func (h *Handler) challenge(c *gin.Context) {
	var req ChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	challenge, err := h.service.GetChallenge(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ChallengeResponse{
		Challenge: challenge,
		TTL:       300,
	})
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	resp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	resp, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) logout(c *gin.Context) {
	deviceIDValue, exists := c.Get(ContextDeviceIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing device context"})
		return
	}

	deviceID, ok := deviceIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid device context"})
		return
	}

	if err := h.service.Logout(c.Request.Context(), deviceID); err != nil {
		h.respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) me(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": profile})
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	profile, err := h.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": profile})
}

func (h *Handler) devices(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	deviceID, err := DeviceIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing device context"})
		return
	}

	devices, err := h.service.ListDevices(c.Request.Context(), userID, deviceID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

func (h *Handler) registerPendingDevice(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	var req RegisterPendingDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	device, err := h.service.RegisterPendingDevice(c.Request.Context(), userID, req)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"device": device})
}

func (h *Handler) revokeDevice(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	currentDeviceID, err := DeviceIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing device context"})
		return
	}

	targetDeviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_device_id", "message": "invalid device id"})
		return
	}

	if err := h.service.RevokeDevice(c.Request.Context(), userID, currentDeviceID, targetDeviceID); err != nil {
		h.respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) approveDevice(c *gin.Context) {
	userID, err := UserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing user context"})
		return
	}

	currentDeviceID, err := DeviceIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing device context"})
		return
	}

	var req ApproveDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	targetDeviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_device_id", "message": "invalid device id"})
		return
	}

	if err := h.service.ApproveDevice(c.Request.Context(), userID, currentDeviceID, targetDeviceID); err != nil {
		h.respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials", "message": err.Error()})
	case errors.Is(err, ErrChallengeExpired):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "challenge_expired", "message": err.Error()})
	case errors.Is(err, ErrRefreshRevoked):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh_revoked", "message": err.Error()})
	case errors.Is(err, ErrInvalidDevice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_device_request", "message": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_error", "message": err.Error()})
	}
}
