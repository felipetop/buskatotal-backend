package http

import (
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, userHandler *UserHandler, authHandler *AuthHandler, infocarHandler *InfocarHandler, paymentHandler *PaymentHandler, authMiddleware *AuthMiddleware, catalogHandler *CatalogHandler, infovistHandler *InfovistHandler, adminHandler *AdminHandler, apifullHandler *ApiFullHandler, lgpdHandler *LGPDHandler) {

    // Rate limit: 10 requests per minute on auth endpoints
    authLimiter := NewRateLimiter(10, 1*time.Minute)

    if authHandler != nil {
        auth := router.Group("/auth")
        {
            auth.POST("/register", authLimiter.Handler(), authHandler.Register)
            auth.POST("/login", authLimiter.Handler(), authHandler.Login)
            auth.GET("/verify-email", authHandler.VerifyEmail)
            auth.POST("/forgot-password", authLimiter.Handler(), authHandler.ForgotPassword)
            auth.POST("/reset-password", authLimiter.Handler(), authHandler.ResetPassword)
        }
        if authMiddleware != nil {
            authProtected := auth.Group("")
            authProtected.Use(authMiddleware.Handler())
            authProtected.POST("/resend-verification", authHandler.ResendVerification)
        }
    }

    users := router.Group("/users")
    {
        if authMiddleware != nil {
            users.GET("/:id/balance", authMiddleware.Handler(), userHandler.GetBalance)
            if lgpdHandler != nil {
                users.GET("/:id/data", authMiddleware.Handler(), lgpdHandler.GetUserData)
                users.GET("/:id/data/export", authMiddleware.Handler(), lgpdHandler.ExportUserData)
                users.POST("/:id/data/deletion-request", authMiddleware.Handler(), lgpdHandler.RequestDeletion)
            }
        } else {
            users.GET("/:id/balance", userHandler.GetBalance)
        }
    }

    if paymentHandler != nil {
        payments := router.Group("/payments")

        // Webhook is public — PicPay calls it with no user token.
        payments.POST("/webhook", paymentHandler.Webhook)

        if authMiddleware != nil {
            protected := payments.Group("")
            protected.Use(authMiddleware.Handler())
            {
                protected.POST("/users/:id/credit", paymentHandler.Credit)
                protected.POST("/users/:id/orders", paymentHandler.CreateOrder)
                protected.GET("/users/:id/orders", paymentHandler.ListOrders)
                protected.POST("/orders/:reference_id/sync", paymentHandler.SyncOrder)
            }
        } else {
            payments.POST("/users/:id/credit", paymentHandler.Credit)
            payments.POST("/users/:id/orders", paymentHandler.CreateOrder)
            payments.GET("/users/:id/orders", paymentHandler.ListOrders)
            payments.POST("/orders/:reference_id/sync", paymentHandler.SyncOrder)
        }
    }

    if catalogHandler != nil {
        router.GET("/catalog", catalogHandler.GetCatalog)
    }

    if infocarHandler != nil {
        consultas := router.Group("/consultas")
        if authMiddleware != nil {
            consultas.Use(authMiddleware.Handler())
        }
        {
            consultas.GET("/veicular/agregados/:tipo/:valor", infocarHandler.GetAgregadosB)
            consultas.GET("/veicular/:produto/:tipo/:valor", infocarHandler.QueryProduct)
        }
    }

    if infovistHandler != nil {
        vistorias := router.Group("/vistorias")
        if authMiddleware != nil {
            vistorias.Use(authMiddleware.Handler())
        }
        {
            vistorias.GET("", infovistHandler.ListInspections)
            vistorias.POST("", infovistHandler.CreateInspection)
            vistorias.GET("/:protocol", infovistHandler.ViewInspection)
            vistorias.GET("/:protocol/relatorio", infovistHandler.GetReportV1)
            vistorias.GET("/:protocol/relatorio-completo", infovistHandler.GetReportV2)
        }
    }

    if apifullHandler != nil {
        apifull := router.Group("/consultas/dados")
        if authMiddleware != nil {
            apifull.Use(authMiddleware.Handler())
        }
        {
            apifull.POST("/:produto", apifullHandler.QueryProduct)
        }
    }

    if adminHandler != nil && authMiddleware != nil {
        admin := router.Group("/admin")
        admin.Use(authMiddleware.Handler())
        admin.Use(AdminOnly())
        {
            admin.GET("/users", adminHandler.ListUsers)
            admin.GET("/users/:id", adminHandler.GetUser)
            if lgpdHandler != nil {
                admin.GET("/deletion-requests", lgpdHandler.ListDeletionRequests)
                admin.PATCH("/deletion-requests/:id", lgpdHandler.ProcessDeletion)
            }
        }
    }
}