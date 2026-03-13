package app

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/configs"
    "buskatotal-backend/internal/infra/firestore"
    "buskatotal-backend/internal/infra/memory"
    httpinterfaces "buskatotal-backend/internal/interfaces/http"
)

func Run() error {
    cfg := configs.Load()

    router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    if cfg.UseMockDB || cfg.FirebaseProjectID == "" {
        userRepo := memory.NewUserRepository()
        taskRepo := memory.NewTaskRepository()
        userService := NewUserService(userRepo)
        taskService := NewTaskService(taskRepo)

        userHandler := httpinterfaces.NewUserHandler(userService)
        taskHandler := httpinterfaces.NewTaskHandler(taskService)
        httpinterfaces.RegisterRoutes(router, userHandler, taskHandler)
    } else {
        client, err := firestore.NewClient(cfg.FirebaseProjectID)
        if err != nil {
            return fmt.Errorf("firestore client: %w", err)
        }
        defer client.Close()

        userRepo := firestore.NewUserRepository(client)
        taskRepo := firestore.NewTaskRepository(client)
        userService := NewUserService(userRepo)
        taskService := NewTaskService(taskRepo)

        userHandler := httpinterfaces.NewUserHandler(userService)
        taskHandler := httpinterfaces.NewTaskHandler(taskService)
        httpinterfaces.RegisterRoutes(router, userHandler, taskHandler)
    }

    addr := fmt.Sprintf(":%s", cfg.Port)
    return router.Run(addr)
}