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

    if paymentHandler != nil {
        payments := router.Group("/payments")
        if authMiddleware != nil {
            payments.Use(authMiddleware.Handler())
        }
        {
            payments.POST("/users/:id/credit", paymentHandler.Credit)
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