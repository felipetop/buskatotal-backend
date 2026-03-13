package configs

import (
    "os"
    "strings"
)

type Config struct {
    Port              string
    FirebaseProjectID string
    FirebaseCredsPath string
    UseMockDB         bool
}

func Load() Config {
    return Config{
        Port:              getEnv("PORT", "8080"),
        FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
        FirebaseCredsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
        UseMockDB:         parseBool(getEnv("USE_MOCK_DB", "false")),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func parseBool(value string) bool {
    normalized := strings.TrimSpace(strings.ToLower(value))
    return normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "y"
}