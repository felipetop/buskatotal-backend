package configs

import "os"

type Config struct {
    Port              string
    FirebaseProjectID string
    FirebaseCredsPath string
}

func Load() Config {
    return Config{
        Port:              getEnv("PORT", "8080"),
        FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
        FirebaseCredsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}