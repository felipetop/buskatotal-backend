package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine, userHandler *UserHandler, taskHandler *TaskHandler, infocarHandler *InfocarHandler, paymentHandler *PaymentHandler) {

    users := router.Group("/users")
    {
        users.POST("", userHandler.Create)
        users.GET("", userHandler.List)
        users.GET("/:id", userHandler.GetByID)
        users.PUT("/:id", userHandler.Update)
        users.DELETE("/:id", userHandler.Delete)
    }

    tasks := router.Group("/tasks")
    {
        tasks.POST("", taskHandler.Create)
        tasks.GET("/:id", taskHandler.GetByID)
        tasks.GET("", taskHandler.ListByUser)
        tasks.PUT("/:id", taskHandler.Update)
        tasks.DELETE("/:id", taskHandler.Delete)
    }

    if paymentHandler != nil {
        payments := router.Group("/payments")
        {
            payments.POST("/users/:id/credit", paymentHandler.Credit)
        }
    }

    if infocarHandler != nil {
        infocar := router.Group("/infocar")
        {
            infocar.GET("/agregados-b/:tipo/:valor", infocarHandler.GetAgregadosB)
        }
    }
}