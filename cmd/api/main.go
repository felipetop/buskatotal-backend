package main

import (
    "log"

    "buskatotal-backend/internal/app"
)

func main() {
    if err := app.Run(); err != nil {
        log.Fatalf("failed to start server: %v", err)
    }
}