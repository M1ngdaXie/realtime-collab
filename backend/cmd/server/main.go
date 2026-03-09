package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net"
    "net/http/pprof"
    "os"
    "strconv"
    "strings"
    "time"

    "runtime"

    "github.com/M1ngdaXie/realtime-collab/internal/auth"
    "github.com/M1ngdaXie/realtime-collab/internal/config"
    "github.com/M1ngdaXie/realtime-collab/internal/handlers"
    "github.com/M1ngdaXie/realtime-collab/internal/hub"
    "github.com/M1ngdaXie/realtime-collab/internal/logger"
    "github.com/M1ngdaXie/realtime-collab/internal/messagebus"
    "github.com/M1ngdaXie/realtime-collab/internal/store"
    "github.com/M1ngdaXie/realtime-collab/internal/workers"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

func main() {
    // CLI flags - override env vars
    portFlag := flag.String("port", "", "Server port (overrides PORT env var)")
    flag.Parse()

    // Load configuration
    cfg, err := config.Load(*portFlag)
    if err != nil {
        log.Fatalf("Configuration error: %v", err)
    }
    log.Printf("Configuration loaded (environment: %s, port: %s)", cfg.Environment, cfg.Port)

    // Initialize structured logger
    zapLogger, err := logger.NewLoggerFromEnv()
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer zapLogger.Sync()

    // Generate unique server ID for this instance
    hostname, _ := os.Hostname()
    serverID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
    zapLogger.Info("Server identity", zap.String("server_id", serverID))

    // Initialize MessageBus (Redis or Local fallback)
    var msgBus messagebus.MessageBus
    if cfg.RedisURL != "" {
        redisBus, err := messagebus.NewRedisMessageBus(cfg.RedisURL, serverID, zapLogger)
        if err != nil {
            zapLogger.Warn("Redis unavailable, falling back to local mode", zap.Error(err))
            msgBus = messagebus.NewLocalMessageBus()
        } else {
            msgBus = redisBus
        }
    } else {
        zapLogger.Info("No REDIS_URL configured, using local mode")
        msgBus = messagebus.NewLocalMessageBus()
    }
    defer msgBus.Close()

    // Initialize database
    dbStore, err := store.NewPostgresStore(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer dbStore.Close()
    log.Println("Database connection established")

    // Initialize WebSocket hub
    wsHub := hub.NewHub(msgBus, serverID, zapLogger)
    go wsHub.Run()
    zapLogger.Info("WebSocket hub started")

    // Start Redis health monitoring (if using Redis)
    if redisBus, ok := msgBus.(*messagebus.RedisMessageBus); ok {
        go redisBus.StartHealthMonitoring(context.Background(), 30*time.Second, func(healthy bool) {
            wsHub.SetFallbackMode(!healthy)
        })
        zapLogger.Info("Redis health monitoring started")
    }

    // Start update persist worker (stream WAL persistence)
    workerCtx, workerCancel := context.WithCancel(context.Background())
    defer workerCancel()
    go workers.StartUpdatePersistWorker(workerCtx, msgBus, dbStore, zapLogger, serverID)
    zapLogger.Info("Update persist worker started")

    // Start periodic session cleanup (every hour)
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        for range ticker.C {
            if err := dbStore.CleanupExpiredSessions(context.Background()); err != nil {
                log.Printf("Error cleaning up expired sessions: %v", err)
            } else {
                log.Println("Cleaned up expired sessions")
            }
        }
    }()
    log.Println("Session cleanup task started")

    // Initialize handlers
    docHandler := handlers.NewDocumentHandler(dbStore, msgBus, serverID, zapLogger)
    wsHandler := handlers.NewWebSocketHandler(wsHub, dbStore, cfg, msgBus)
    authHandler := handlers.NewAuthHandler(dbStore, cfg)
    authMiddleware := auth.NewAuthMiddleware(dbStore, cfg.JWTSecret, zapLogger)
    shareHandler := handlers.NewShareHandler(dbStore, cfg)
    versionHandler := handlers.NewVersionHandler(dbStore)

    // Setup Gin router
    router := gin.Default()

    // Optional pprof endpoints for profiling under load (guarded by env).
    // Enable with: ENABLE_PPROF=1
    // Optional: PPROF_BLOCK_RATE=1 PPROF_MUTEX_FRACTION=1 (adds overhead; use for short profiling windows).
    if shouldEnablePprof(cfg) {
        blockRate := getEnvInt("PPROF_BLOCK_RATE", 0)
        mutexFraction := getEnvInt("PPROF_MUTEX_FRACTION", 0)
        localOnly := getEnvBool("PPROF_LOCAL_ONLY", true)

        if blockRate > 0 {
            runtime.SetBlockProfileRate(blockRate)
        }
        if mutexFraction > 0 {
            runtime.SetMutexProfileFraction(mutexFraction)
        }

        pprofGroup := router.Group("/debug/pprof")
        if localOnly {
            pprofGroup.Use(func(c *gin.Context) {
                ip := net.ParseIP(c.ClientIP())
                if ip == nil || !ip.IsLoopback() {
                    c.AbortWithStatus(403)
                    return
                }
                c.Next()
            })
        }

        user, pass := os.Getenv("PPROF_USER"), os.Getenv("PPROF_PASS")
        if user != "" || pass != "" {
            if user == "" || pass == "" {
                zapLogger.Warn("PPROF_USER/PPROF_PASS must both be set; skipping basic auth")
            } else {
                pprofGroup.Use(gin.BasicAuth(gin.Accounts{user: pass}))
            }
        }

        pprofGroup.GET("/", gin.WrapF(pprof.Index))
        pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
        pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
        pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
        pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))

        pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
        pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
        pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
        pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
        pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
        pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))

        zapLogger.Info("pprof enabled",
            zap.Bool("local_only", localOnly),
            zap.Int("block_rate", blockRate),
            zap.Int("mutex_fraction", mutexFraction),
        )
    }

    // CORS configuration
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowOrigins = cfg.AllowedOrigins
    corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
    corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
    corsConfig.AllowCredentials = true
    router.Use(cors.New(corsConfig))

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // WebSocket endpoint (no auth required, validated in handler)
    router.GET("/ws/:roomId", wsHandler.HandleWebSocket)

    // Load test endpoint - NO AUTH (only for local testing!)
    router.GET("/ws/loadtest/:roomId", wsHandler.HandleWebSocketLoadTest)

    // REST API
    api := router.Group("/api")

    authGroup := api.Group("/auth")
    {
        authGroup.GET("/google", authHandler.GoogleLogin)
        authGroup.GET("/google/callback", authHandler.GoogleCallback)
        authGroup.GET("/github", authHandler.GithubLogin)
        authGroup.GET("/github/callback", authHandler.GithubCallback)
        authGroup.GET("/me", authMiddleware.RequireAuth(), authHandler.Me)
        authGroup.POST("/logout", authMiddleware.RequireAuth(), authHandler.Logout)
    }

    // Document routes with optional auth
    docs := api.Group("/documents")

    {
        docs.GET("", authMiddleware.RequireAuth(), docHandler.ListDocuments)
        docs.GET("/:id", authMiddleware.RequireAuth(), docHandler.GetDocument)
        docs.GET("/:id/state", authMiddleware.OptionalAuth(), docHandler.GetDocumentState)

        // Permission route (supports both auth and share token)
        docs.GET("/:id/permission", authMiddleware.OptionalAuth(), docHandler.GetDocumentPermission)

        docs.POST("", authMiddleware.RequireAuth(), docHandler.CreateDocument)
        docs.PUT("/:id/state", authMiddleware.RequireAuth(), docHandler.UpdateDocumentState)
        docs.DELETE("/:id", authMiddleware.RequireAuth(), docHandler.DeleteDocument)

        // Share routes
        docs.POST("/:id/shares", authMiddleware.RequireAuth(), shareHandler.CreateShare)
        docs.GET("/:id/shares", authMiddleware.RequireAuth(), shareHandler.ListShares)
        docs.DELETE("/:id/shares/:userId", authMiddleware.RequireAuth(), shareHandler.DeleteShare)
        docs.POST("/:id/share-link", authMiddleware.RequireAuth(), shareHandler.CreateShareLink)
        docs.GET("/:id/share-link", authMiddleware.RequireAuth(), shareHandler.GetShareLink)
        docs.DELETE("/:id/share-link", authMiddleware.RequireAuth(), shareHandler.RevokeShareLink)

        // Version history routes
        docs.POST("/:id/versions", authMiddleware.RequireAuth(), versionHandler.CreateVersion)
        docs.GET("/:id/versions", authMiddleware.RequireAuth(), versionHandler.ListVersions)
        docs.GET("/:id/versions/:versionId/snapshot", authMiddleware.RequireAuth(), versionHandler.GetVersionSnapshot)
        docs.POST("/:id/restore", authMiddleware.RequireAuth(), versionHandler.RestoreVersion)
    }

    // Start server
    log.Printf("Server starting on port %s", cfg.Port)
    if err := router.Run(":" + cfg.Port); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

func shouldEnablePprof(cfg *config.Config) bool {
    if cfg == nil || cfg.IsProduction() {
        return false
    }
    return getEnvBool("ENABLE_PPROF", false)
}

func getEnvBool(key string, defaultValue bool) bool {
    value, ok := os.LookupEnv(key)
    if !ok {
        return defaultValue
    }

    switch strings.ToLower(strings.TrimSpace(value)) {
    case "1", "true", "t", "yes", "y", "on":
        return true
    case "0", "false", "f", "no", "n", "off":
        return false
    default:
        return defaultValue
    }
}

func getEnvInt(key string, defaultValue int) int {
    value, ok := os.LookupEnv(key)
    if !ok {
        return defaultValue
    }
    parsed, err := strconv.Atoi(strings.TrimSpace(value))
    if err != nil {
        return defaultValue
    }
    return parsed
}
