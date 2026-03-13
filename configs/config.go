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
    InfocarIDKey       string
    InfocarUser        string
    InfocarPassword    string
    InfocarBaseURL     string
}

func Load() Config {
    return Config{
        Port:              getEnv("PORT", "8080"),
        FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
        FirebaseCredsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
        UseMockDB:         parseBool(getEnv("USE_MOCK_DB", "false")),
        // INFOCAR_ID_KEY, INFOCAR_USER, INFOCAR_PASSWORD devem ser definidos em variáveis
        // de ambiente seguras (ex.: secret manager). Não comitar valores reais no código.
        InfocarIDKey:   getEnv("INFOCAR_ID_KEY", ""),
        InfocarUser:    getEnv("INFOCAR_USER", ""),
        InfocarPassword: getEnv("INFOCAR_PASSWORD", ""),
        InfocarBaseURL: getEnv("INFOCAR_BASE_URL", "https://api.datacast3.com/api"),
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