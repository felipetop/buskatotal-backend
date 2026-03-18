package auth

import (
    "context"
    "errors"

    jwt "github.com/golang-jwt/jwt/v5"

    domain "buskatotal-backend/internal/domain/auth"
)

type JWTProvider struct {
    secret []byte
}

func NewJWTProvider(secret string) *JWTProvider {
    return &JWTProvider{secret: []byte(secret)}
}

func (p *JWTProvider) Authenticate(ctx context.Context, token string) (domain.Result, error) {
    parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return p.secret, nil
    })
    if err != nil || !parsed.Valid {
        return domain.Result{}, errors.New("invalid token")
    }

    claims, ok := parsed.Claims.(jwt.MapClaims)
    if !ok {
        return domain.Result{}, errors.New("invalid claims")
    }

    userID, ok := claims["userId"].(string)
    if !ok || userID == "" {
        return domain.Result{}, errors.New("userId claim missing")
    }

    role, _ := claims["role"].(string)
    if role == "" {
        role = "user"
    }

    return domain.Result{UserID: userID, Role: role}, nil
}
