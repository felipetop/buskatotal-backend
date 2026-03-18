package http

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/domain/auth"
)

const authUserIDKey = "authUserID"
const authUserRoleKey = "authUserRole"

type AuthMiddleware struct {
    provider auth.Provider
    header   string
}

// AuthProvider re-export to simplify wiring in app layer.
type AuthProvider = auth.Provider

func NewAuthMiddleware(provider auth.Provider, header string) *AuthMiddleware {
    return &AuthMiddleware{provider: provider, header: header}
}

func (m *AuthMiddleware) Handler() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader(m.header)
        token = strings.TrimSpace(token)
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth token"})
            return
        }

        // For JWT, the header should be Authorization: Bearer <token>
        if strings.HasPrefix(strings.ToLower(token), "bearer ") {
            token = strings.TrimSpace(token[len("bearer "):])
        }

        result, err := m.provider.Authenticate(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            return
        }

        c.Set(authUserIDKey, result.UserID)
        c.Set(authUserRoleKey, result.Role)
        c.Next()
    }
}

func GetAuthUserID(c *gin.Context) (string, bool) {
    value, ok := c.Get(authUserIDKey)
    if !ok {
        return "", false
    }
    userID, ok := value.(string)
    return userID, ok
}

func GetAuthUserRole(c *gin.Context) string {
    value, ok := c.Get(authUserRoleKey)
    if !ok {
        return "user"
    }
    role, ok := value.(string)
    if !ok {
        return "user"
    }
    return role
}

// AdminOnly returns a middleware that rejects non-admin users.
func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        role := GetAuthUserRole(c)
        if role != "admin" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
            return
        }
        c.Next()
    }
}