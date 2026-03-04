package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	jwtManager := NewJWTManager("test-secret", "local.naier", 15*time.Minute, 24*time.Hour)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["message"] != "missing authorization header" {
		t.Fatalf("expected missing authorization header message, got %q", body["message"])
	}
}

func TestAuthMiddlewareRejectsRefreshToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	jwtManager := NewJWTManager("test-secret", "local.naier", 15*time.Minute, 24*time.Hour)

	userID := uuid.New()
	deviceID := uuid.New()
	_, refreshToken, err := jwtManager.GenerateTokenPair(userID, deviceID)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}

	router := gin.New()
	router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["message"] != "access token required" {
		t.Fatalf("expected access token required message, got %q", body["message"])
	}
}

func TestAuthMiddlewareAcceptsAccessTokenAndSetsContext(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	jwtManager := NewJWTManager("test-secret", "local.naier", 15*time.Minute, 24*time.Hour)

	userID := uuid.New()
	deviceID := uuid.New()
	accessToken, _, err := jwtManager.GenerateTokenPair(userID, deviceID)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}

	router := gin.New()
	router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		gotUserID, err := UserIDFromContext(c)
		if err != nil {
			t.Fatalf("user id from context: %v", err)
		}
		gotDeviceID, err := DeviceIDFromContext(c)
		if err != nil {
			t.Fatalf("device id from context: %v", err)
		}

		if gotUserID != userID {
			t.Fatalf("expected user id %s, got %s", userID, gotUserID)
		}
		if gotDeviceID != deviceID {
			t.Fatalf("expected device id %s, got %s", deviceID, gotDeviceID)
		}

		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidUserClaim(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	jwtManager := NewJWTManager("test-secret", "local.naier", 15*time.Minute, 24*time.Hour)

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID:    "not-a-uuid",
		DeviceID:  uuid.NewString(),
		ServerID:  "local.naier",
		TokenType: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "local.naier",
			Subject:   "not-a-uuid",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	router := gin.New()
	router.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}
