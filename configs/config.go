package configs

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	FirebaseProjectID string
	FirebaseCredsPath string
	UseMockDB         bool
	InfocarIDKey      string
	InfocarUser       string
	InfocarPassword   string
	InfocarBaseURL    string
	AuthMode          string
	AuthHeader        string
	AuthJWTSecret     string
	PicPayClientID     string  // OAuth2 client_id for PicPay Checkout API
	PicPayClientSecret string  // OAuth2 client_secret for PicPay Checkout API
	AppBaseURL         string  // public base URL used to build the PicPay callback URL
	CatalogMarkup      float64 // sale price multiplier applied to cost prices (e.g. 3.0 = 3x cost)
}

func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		FirebaseProjectID: getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		UseMockDB:         parseBool(getEnv("USE_MOCK_DB", "false")),
		// INFOCAR_ID_KEY, INFOCAR_USER, INFOCAR_PASSWORD devem ser definidos em variáveis
		// de ambiente seguras (ex.: secret manager). Não comitar valores reais no código.
		InfocarIDKey:       getEnv("INFOCAR_ID_KEY", ""),
		InfocarUser:        getEnv("INFOCAR_USER", ""),
		InfocarPassword:    getEnv("INFOCAR_PASSWORD", ""),
		InfocarBaseURL:     getEnv("INFOCAR_BASE_URL", "https://api.datacast3.com/api"),
		AuthMode:           getEnv("AUTH_MODE", "mock"),
		AuthHeader:         getEnv("AUTH_HEADER", "X-User-Id"),
		AuthJWTSecret:      getEnv("AUTH_JWT_SECRET", ""),
		PicPayClientID:     getEnv("PICPAY_CLIENT_ID", ""),
		PicPayClientSecret: getEnv("PICPAY_CLIENT_SECRET", ""),
		AppBaseURL:         getEnv("APP_BASE_URL", "http://localhost:8080"),
		CatalogMarkup:      parseFloat(getEnv("CATALOG_MARKUP", "2.0")),
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

func parseFloat(value string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || f <= 0 {
		return 3.0
	}
	return f
}
