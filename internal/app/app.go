package app

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/configs"
    "buskatotal-backend/internal/domain/payment"
    "buskatotal-backend/internal/infra/firestore"
    "buskatotal-backend/internal/infra/infocar"
    "buskatotal-backend/internal/infra/memory"
    authinfra "buskatotal-backend/internal/infra/auth"
    paymentinfra "buskatotal-backend/internal/infra/payment"
    httpinterfaces "buskatotal-backend/internal/interfaces/http"
)

func Run() error {
    cfg := configs.Load()

    if cfg.AuthJWTSecret == "" {
        cfg.AuthJWTSecret = "dev-secret"
    }

    router := gin.Default()
    router.Use(func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-Id")
        if c.Request.Method == http.MethodOptions {
            c.Status(http.StatusNoContent)
            return
        }
        c.Next()
    })
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    var authProvider httpinterfaces.AuthProvider
    if cfg.AuthMode == "jwt" {
        authProvider = authinfra.NewJWTProvider(cfg.AuthJWTSecret)
    } else {
        authProvider = authinfra.NewMockProvider(cfg.AuthHeader)
    }
    authMiddleware := httpinterfaces.NewAuthMiddleware(authProvider, cfg.AuthHeader)

    // Select payment provider: use PicPay when a token is configured, mock otherwise.
    var paymentProvider payment.Provider
    if cfg.PicPayClientID != "" && cfg.PicPayClientSecret != "" {
        paymentProvider = paymentinfra.NewPicPayProvider(cfg.PicPayClientID, cfg.PicPayClientSecret)
    } else {
        paymentProvider = paymentinfra.NewMockProvider()
    }

    if cfg.UseMockDB || cfg.FirebaseProjectID == "" {
        userRepo := memory.NewUserRepository()
        orderRepo := memory.NewOrderRepository()
        userService := NewUserService(userRepo)
        authService := NewAuthService(userRepo, cfg.AuthJWTSecret, 24*time.Hour)
        infocarClient := infocar.NewClient(cfg.InfocarBaseURL, cfg.InfocarIDKey, cfg.InfocarUser, cfg.InfocarPassword)
        infocarService := NewInfocarService(infocarClient, userRepo, 150)
        infocarHandler := httpinterfaces.NewInfocarHandler(infocarService)
        paymentService := NewPaymentService(paymentProvider, orderRepo, userRepo, cfg.AppBaseURL)
        paymentHandler := httpinterfaces.NewPaymentHandler(paymentService)

        userHandler := httpinterfaces.NewUserHandler(userService)
        authHandler := httpinterfaces.NewAuthHandler(authService)
        httpinterfaces.RegisterRoutes(router, userHandler, authHandler, infocarHandler, paymentHandler, authMiddleware)
    } else {
        client, err := firestore.NewClient(cfg.FirebaseProjectID)
        if err != nil {
            return fmt.Errorf("firestore client: %w", err)
        }
        defer client.Close()

        userRepo := firestore.NewUserRepository(client)
        orderRepo := firestore.NewOrderRepository(client)
        userService := NewUserService(userRepo)
        authService := NewAuthService(userRepo, cfg.AuthJWTSecret, 24*time.Hour)
        infocarClient := infocar.NewClient(cfg.InfocarBaseURL, cfg.InfocarIDKey, cfg.InfocarUser, cfg.InfocarPassword)
        infocarService := NewInfocarService(infocarClient, userRepo, 150)
        infocarHandler := httpinterfaces.NewInfocarHandler(infocarService)
        paymentService := NewPaymentService(paymentProvider, orderRepo, userRepo, cfg.AppBaseURL)
        paymentHandler := httpinterfaces.NewPaymentHandler(paymentService)

        go startReconciliationWorker(paymentService)

        userHandler := httpinterfaces.NewUserHandler(userService)
        authHandler := httpinterfaces.NewAuthHandler(authService)
        httpinterfaces.RegisterRoutes(router, userHandler, authHandler, infocarHandler, paymentHandler, authMiddleware)
    }

    addr := fmt.Sprintf(":%s", cfg.Port)
    return router.Run(addr)
}

func startReconciliationWorker(svc *PaymentService) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        log.Println("reconciliation: checking pending orders...")
        svc.ReconcileOrders(context.Background())
    }
}