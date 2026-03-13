package app

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/configs"
    "buskatotal-backend/internal/infra/firestore"
    httpinterfaces "buskatotal-backend/internal/interfaces/http"
)

func Run() error {
    cfg := configs.Load()

    if cfg.FirebaseProjectID == "" {
        return fmt.Errorf("FIREBASE_PROJECT_ID is required")
    }

    client, err := firestore.NewClient(cfg.FirebaseProjectID)
    if err != nil {
        return fmt.Errorf("firestore client: %w", err)
    }
    defer client.Close()

    router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    httpinterfaces.RegisterRoutes(router, client)

    addr := fmt.Sprintf(":%s", cfg.Port)
    return router.Run(addr)
}