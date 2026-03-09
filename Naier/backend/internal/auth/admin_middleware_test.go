package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminTokenMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("accepts matching token", func(t *testing.T) {
		router := gin.New()
		router.Use(AdminTokenMiddleware("secret-token"))
		router.GET("/admin", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("X-Admin-Token", "secret-token")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", recorder.Code)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		router := gin.New()
		router.Use(AdminTokenMiddleware("secret-token"))
		router.GET("/admin", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", recorder.Code)
		}
	})
}
