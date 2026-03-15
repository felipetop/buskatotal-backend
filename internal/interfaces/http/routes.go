package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine, userHandler *UserHandler, authHandler *AuthHandler, infocarHandler *InfocarHandler, paymentHandler *PaymentHandler, authMiddleware *AuthMiddleware) {

    if authHandler != nil {
        auth := router.Group("/auth")
        {
            auth.POST("/register", authHandler.Register)
            auth.POST("/login", authHandler.Login)
        }
    }

    users := router.Group("/users")
    {
        users.POST("", userHandler.Create)
        users.GET("", userHandler.List)
        users.GET("/:id", userHandler.GetByID)
        users.PUT("/:id", userHandler.Update)
        users.DELETE("/:id", userHandler.Delete)
    }

    // Balance endpoint — protected, user can only query their own balance.
    if authMiddleware != nil {
        balanceGroup := router.Group("/users")
        balanceGroup.Use(authMiddleware.Handler())
        balanceGroup.GET("/:id/balance", userHandler.GetBalance)
    } else {
        router.GET("/users/:id/balance", userHandler.GetBalance)
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
            }
        } else {
            payments.POST("/users/:id/credit", paymentHandler.Credit)
            payments.POST("/users/:id/orders", paymentHandler.CreateOrder)
            payments.GET("/users/:id/orders", paymentHandler.ListOrders)
        }
    }

    if infocarHandler != nil {
        infocar := router.Group("/infocar")
        if authMiddleware != nil {
            infocar.Use(authMiddleware.Handler())
        }
        {
            infocar.GET("/agregados-b/:tipo/:valor", infocarHandler.GetAgregadosB)
        }
    }
}