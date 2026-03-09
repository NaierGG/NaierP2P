package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminTokenMiddleware(apiToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(apiToken) == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "admin_api_disabled",
				"message": "admin api token is not configured",
			})
			return
		}

		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" {
			authorization := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
				token = strings.TrimSpace(authorization[7:])
			}
		}

		if subtleCompare(token, apiToken) == false {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "admin_unauthorized",
				"message": "missing or invalid admin token",
			})
			return
		}

		c.Next()
	}
}
