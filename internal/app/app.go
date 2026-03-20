package app

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/configs"
    "buskatotal-backend/internal/domain/email"
    "buskatotal-backend/internal/domain/payment"
    "buskatotal-backend/internal/infra/firestore"
    "buskatotal-backend/internal/infra/apifull"
    "buskatotal-backend/internal/infra/infocar"
    "buskatotal-backend/internal/infra/infovist"
    "buskatotal-backend/internal/infra/memory"
    "buskatotal-backend/internal/infra/resend"
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
    isMockPayment := cfg.PicPayClientID == "" || cfg.PicPayClientSecret == ""
    if !isMockPayment {
        paymentProvider = paymentinfra.NewPicPayProvider(cfg.PicPayClientID, cfg.PicPayClientSecret)
    } else {
        paymentProvider = paymentinfra.NewMockProvider()
    }

    var emailVerificationService *EmailVerificationService
    var emailSender email.Sender
    if cfg.ResendAPIKey != "" {
        emailSender = resend.NewClient(cfg.ResendAPIKey)
    }

    if cfg.UseMockDB || cfg.FirebaseProjectID == "" {
        userRepo := memory.NewUserRepository()
        orderRepo := memory.NewOrderRepository()
        inspRepo := memory.NewInspectionRepository()
        deletionRepo := memory.NewDeletionRepository()
        logRepo := memory.NewLogRepository()
        if emailSender != nil {
            verificationRepo := memory.NewVerificationRepository()
            emailVerificationService = NewEmailVerificationService(verificationRepo, userRepo, emailSender)
        }
        userService := NewUserService(userRepo)
        authService := NewAuthService(userRepo, cfg.AuthJWTSecret, 24*time.Hour, emailVerificationService)
        lgpdService := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, emailSender)
        infocarClient := infocar.NewClient(cfg.InfocarBaseURL, cfg.InfocarIDKey, cfg.InfocarUser, cfg.InfocarPassword)
        infocarService := NewInfocarService(infocarClient, userRepo, 150)
        infocarHandler := httpinterfaces.NewInfocarHandler(infocarService)
        paymentService := NewPaymentService(paymentProvider, orderRepo, userRepo, cfg.AppBaseURL)
        paymentHandler := httpinterfaces.NewPaymentHandler(paymentService, isMockPayment)

        infovistClient := infovist.NewClient(cfg.InfovistBaseURL, cfg.InfovistEmail, cfg.InfovistPassword, cfg.InfovistAPIToken)
        infovistService := // Custo de venda: INFOVIST (R$10,32) + VISTORIA DIGITAL (R$34,52) = R$44,84 custo × 3x markup = R$134,52 = 13452 centavos
		NewInfovistService(infovistClient, userRepo, inspRepo, 13452, 0)
        infovistHandler := httpinterfaces.NewInfovistHandler(infovistService)

        adminService := NewAdminService(userRepo)
        adminHandler := httpinterfaces.NewAdminHandler(adminService)

        userHandler := httpinterfaces.NewUserHandler(userService)
        authHandler := httpinterfaces.NewAuthHandler(authService, asEmailVerifier(emailVerificationService))
        lgpdHandler := httpinterfaces.NewLGPDHandler(lgpdService)
        apifullClient := apifull.NewClient(cfg.ApiFullBaseURL, cfg.ApiFullToken)
        apifullService := NewApiFullService(apifullClient, userRepo)
        apifullHandler := httpinterfaces.NewApiFullHandler(apifullService)

        catalogHandler := httpinterfaces.NewCatalogHandler(cfg.CatalogMarkup)
        httpinterfaces.RegisterRoutes(router, userHandler, authHandler, infocarHandler, paymentHandler, authMiddleware, catalogHandler, infovistHandler, adminHandler, apifullHandler, lgpdHandler)
    } else {
        client, err := firestore.NewClient(cfg.FirebaseProjectID)
        if err != nil {
            return fmt.Errorf("firestore client: %w", err)
        }
        defer client.Close()

        userRepo := firestore.NewUserRepository(client)
        orderRepo := firestore.NewOrderRepository(client)
        inspRepo := firestore.NewInspectionRepository(client)
        deletionRepo := firestore.NewDeletionRepository(client)
        logRepo := firestore.NewLogRepository(client)
        if emailSender != nil {
            verificationRepo := firestore.NewVerificationRepository(client)
            emailVerificationService = NewEmailVerificationService(verificationRepo, userRepo, emailSender)
        }
        userService := NewUserService(userRepo)
        authService := NewAuthService(userRepo, cfg.AuthJWTSecret, 24*time.Hour, emailVerificationService)
        lgpdService := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, emailSender)
        infocarClient := infocar.NewClient(cfg.InfocarBaseURL, cfg.InfocarIDKey, cfg.InfocarUser, cfg.InfocarPassword)
        infocarService := NewInfocarService(infocarClient, userRepo, 150)
        infocarHandler := httpinterfaces.NewInfocarHandler(infocarService)
        paymentService := NewPaymentService(paymentProvider, orderRepo, userRepo, cfg.AppBaseURL)
        paymentHandler := httpinterfaces.NewPaymentHandler(paymentService, isMockPayment)

        infovistClient := infovist.NewClient(cfg.InfovistBaseURL, cfg.InfovistEmail, cfg.InfovistPassword, cfg.InfovistAPIToken)
        infovistService := // Custo de venda: INFOVIST (R$10,32) + VISTORIA DIGITAL (R$34,52) = R$44,84 custo × 3x markup = R$134,52 = 13452 centavos
		NewInfovistService(infovistClient, userRepo, inspRepo, 13452, 0)
        infovistHandler := httpinterfaces.NewInfovistHandler(infovistService)

        go startReconciliationWorker(paymentService)

        adminService := NewAdminService(userRepo)
        adminHandler := httpinterfaces.NewAdminHandler(adminService)

        userHandler := httpinterfaces.NewUserHandler(userService)
        authHandler := httpinterfaces.NewAuthHandler(authService, asEmailVerifier(emailVerificationService))
        lgpdHandler := httpinterfaces.NewLGPDHandler(lgpdService)
        apifullClient := apifull.NewClient(cfg.ApiFullBaseURL, cfg.ApiFullToken)
        apifullService := NewApiFullService(apifullClient, userRepo)
        apifullHandler := httpinterfaces.NewApiFullHandler(apifullService)

        catalogHandler := httpinterfaces.NewCatalogHandler(cfg.CatalogMarkup)
        httpinterfaces.RegisterRoutes(router, userHandler, authHandler, infocarHandler, paymentHandler, authMiddleware, catalogHandler, infovistHandler, adminHandler, apifullHandler, lgpdHandler)
    }

    addr := fmt.Sprintf(":%s", cfg.Port)
    return router.Run(addr)
}

// asEmailVerifier converts a possibly-nil *EmailVerificationService to a properly nil interface.
func asEmailVerifier(svc *EmailVerificationService) httpinterfaces.EmailVerificationService {
    if svc == nil {
        return nil
    }
    return svc
}

func startReconciliationWorker(svc *PaymentService) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        log.Println("reconciliation: checking pending orders...")
        svc.ReconcileOrders(context.Background())
    }
}