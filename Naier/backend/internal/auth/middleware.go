package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextUserIDKey   = "userID"
	ContextDeviceIDKey = "deviceID"
)

func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "missing authorization header"})
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid bearer token"})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid bearer token"})
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid token"})
			return
		}
		if claims.TokenType != tokenTypeAccess {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "access token required"})
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid user claim"})
			return
		}

		deviceID, err := uuid.Parse(claims.DeviceID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "invalid device claim"})
			return
		}

		c.Set(ContextUserIDKey, userID)
		c.Set(ContextDeviceIDKey, deviceID)
		c.Next()
	}
}

func UserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(ContextUserIDKey)
	if !exists {
		return uuid.Nil, errors.New("user id missing from context")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user id has invalid type")
	}

	return userID, nil
}

func DeviceIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(ContextDeviceIDKey)
	if !exists {
		return uuid.Nil, errors.New("device id missing from context")
	}

	deviceID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("device id has invalid type")
	}

	return deviceID, nil
}
