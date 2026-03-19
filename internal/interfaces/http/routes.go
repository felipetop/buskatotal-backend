package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine, userHandler *UserHandler, authHandler *AuthHandler, infocarHandler *InfocarHandler, paymentHandler *PaymentHandler, authMiddleware *AuthMiddleware, catalogHandler *CatalogHandler, infovistHandler *InfovistHandler, adminHandler *AdminHandler, apifullHandler *ApiFullHandler) {

    if authHandler != nil {
        auth := router.Group("/auth")
        {
            auth.POST("/register", authHandler.Register)
            auth.POST("/login", authHandler.Login)
        }
    }

    users := router.Group("/users")
    {
        if authMiddleware != nil {
            users.GET("/:id/balance", authMiddleware.Handler(), userHandler.GetBalance)
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
        }
    }
}