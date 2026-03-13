package http

import (
    "cloud.google.com/go/firestore"
    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/app"
    firestoreinfra "buskatotal-backend/internal/infra/firestore"
)

func RegisterRoutes(router *gin.Engine, client *firestore.Client) {
    userRepo := firestoreinfra.NewUserRepository(client)
    taskRepo := firestoreinfra.NewTaskRepository(client)

    userService := app.NewUserService(userRepo)
    taskService := app.NewTaskService(taskRepo)

    userHandler := NewUserHandler(userService)
    taskHandler := NewTaskHandler(taskService)

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
}