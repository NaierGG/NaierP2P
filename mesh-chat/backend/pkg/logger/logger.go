package logger

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func New(mode string) (*zap.Logger, error) {
	if mode == gin.DebugMode {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}

	return ""
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(string(requestIDKey), requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(WithRequestID(c.Request.Context(), requestID))
		c.Next()
	}
}

func GinLogger(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		requestID, _ := c.Get(string(requestIDKey))
		fields := []zap.Field{
			zap.String("request_id", toString(requestID)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.String("client_ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
			base.Error("request completed with errors", fields...)
			return
		}

		base.Info("request completed", fields...)
	}
}

func Recovery(base *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get(string(requestIDKey))
		base.Error(
			"panic recovered",
			zap.String("request_id", toString(requestID)),
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":      "internal_server_error",
			"message":    "unexpected server error",
			"request_id": requestID,
		})
	})
}

func With(base *zap.Logger, ctx context.Context) *zap.Logger {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return base
	}

	return base.With(zap.String("request_id", requestID))
}

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}

	return ""
}
