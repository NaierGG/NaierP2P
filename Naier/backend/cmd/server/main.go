package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naier/backend/internal/auth"
	"github.com/naier/backend/internal/channel"
	"github.com/naier/backend/internal/config"
	"github.com/naier/backend/internal/federation"
	"github.com/naier/backend/internal/media"
	"github.com/naier/backend/internal/message"
	"github.com/naier/backend/internal/presence"
	appws "github.com/naier/backend/internal/websocket"
	"github.com/naier/backend/pkg/database"
	httplogger "github.com/naier/backend/pkg/logger"
	validatorpkg "github.com/naier/backend/pkg/validator"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	gin.SetMode(cfg.Server.Mode)

	log, err := httplogger.New(cfg.Server.Mode)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := database.NewPostgresPool(ctx, cfg.Database.PostgresDSN)
	if err != nil {
		log.Fatal("postgres connection failed", zap.Error(err))
	}
	defer dbPool.Close()

	redisClient, err := database.NewRedisClient(ctx, cfg.Database.RedisAddr, cfg.Database.RedisPassword)
	if err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}
	defer func() {
		_ = redisClient.Close()
	}()

	router := gin.New()
	router.Use(corsMiddleware(cfg.Server.AllowedOrigins))
	router.Use(httplogger.RequestIDMiddleware())
	router.Use(httplogger.GinLogger(log))
	router.Use(httplogger.Recovery(log))
	router.Use(rateLimitMiddleware(120, time.Minute))

	validate := validatorpkg.New()
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Federation.ServerDomain,
		cfg.Auth.JWTExpiry,
		cfg.Auth.RefreshExpiry,
	)
	authService := auth.NewService(dbPool, redisClient, validate, jwtManager, cfg.Auth.RefreshExpiry, cfg.Beta.InviteOnly)
	authHandler := auth.NewHandler(authService)
	channelRepo := channel.NewRepository(dbPool)
	channelService := channel.NewService(channelRepo, validate)
	channelHandler := channel.NewHandler(channelService)
	messageRepo := message.NewRepository(dbPool)
	messageService := message.NewService(messageRepo, validate)
	messageHandler := message.NewHandler(messageService)
	presenceRepo := presence.NewRepository(redisClient)
	hub := appws.NewHub(redisClient)
	hub.SetDeliveryTracker(messageService)
	presenceService := presence.NewService(presenceRepo, hub)
	wsRouter := appws.NewRouter(hub, jwtManager, messageService, presenceService)
	hub.SetRouter(wsRouter)
	go hub.Run(ctx)

	var mediaStorage *media.Storage
	if cfg.Media.MinIOEndpoint != "" {
		createdStorage, storageErr := media.NewStorage(
			cfg.Media.MinIOEndpoint,
			cfg.Media.MinIOAccessKey,
			cfg.Media.MinIOSecretKey,
			false,
		)
		if storageErr != nil {
			log.Warn("media storage initialization failed", zap.Error(storageErr))
		} else {
			mediaStorage = createdStorage
		}
	}
	mediaService := media.NewService(mediaStorage, cfg.Media.MinIOBucket)
	mediaHandler := media.NewHandler(mediaService)
	federationResolver := federation.NewResolver(dbPool)
	federationService := federation.NewService(dbPool, federationResolver, cfg.Federation)
	federationHandler := federation.NewHandler(federationService)

	router.GET("/health", func(c *gin.Context) {
		healthCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := dbPool.Ping(healthCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "degraded",
				"db":     err.Error(),
			})
			return
		}

		if err := redisClient.Ping(healthCtx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "degraded",
				"redis":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"mode":   cfg.Server.Mode,
		})
	})

	api := router.Group("/api/v1")
	authHandler.RegisterRoutes(api.Group("/auth"), auth.AuthMiddleware(jwtManager))
	adminAPI := api.Group("/admin")
	adminAPI.Use(auth.AdminTokenMiddleware(cfg.Admin.APIToken))
	authHandler.RegisterAdminRoutes(adminAPI)
	api.GET("/ws", wsRouter.ServeWS)
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware(jwtManager))
	channelHandler.RegisterRoutes(protected)
	messageHandler.RegisterRoutes(protected)
	mediaHandler.RegisterRoutes(protected)
	federationHandler.RegisterProtectedRoutes(protected)
	federationHandler.RegisterRoutes(router)

	server := &http.Server{
		Addr:              cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("server starting", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", zap.Error(err))
	}

	log.Info("server stopped")
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientCounter
	limit   int
	window  time.Duration
}

type clientCounter struct {
	count     int
	expiresAt time.Time
}

func rateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := &rateLimiter{
		clients: make(map[string]*clientCounter),
		limit:   limit,
		window:  window,
	}

	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "too many requests",
			})
			return
		}

		c.Next()
	}
}

func (r *rateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.clients[ip]
	if !exists || now.After(entry.expiresAt) {
		r.clients[ip] = &clientCounter{
			count:     1,
			expiresAt: now.Add(r.window),
		}
		return true
	}

	if entry.count >= r.limit {
		return false
	}

	entry.count++
	return true
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "origin_not_allowed",
					"message": "request origin is not allowed",
				})
				return
			}

			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
