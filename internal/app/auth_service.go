package app

import (
    "context"
    "errors"
    "time"

    jwt "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"

    "buskatotal-backend/internal/domain/user"
)

type AuthService struct {
    repo      user.Repository
    jwtSecret []byte
    tokenTTL  time.Duration
}

func NewAuthService(repo user.Repository, jwtSecret string, tokenTTL time.Duration) *AuthService {
    return &AuthService{repo: repo, jwtSecret: []byte(jwtSecret), tokenTTL: tokenTTL}
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (user.User, string, error) {
    if email == "" || password == "" {
        return user.User{}, "", errors.New("email and password are required")
    }

    if name == "" {
        name = email
    }

    if _, err := s.repo.GetByEmail(ctx, email); err == nil {
        return user.User{}, "", errors.New("email already registered")
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return user.User{}, "", errors.New("could not hash password")
    }

    created, err := s.repo.Create(ctx, user.User{
        Name:         name,
        Email:        email,
        PasswordHash: string(hash),
    })
    if err != nil {
        return user.User{}, "", err
    }

    token, err := s.generateToken(created.ID)
    if err != nil {
        return user.User{}, "", err
    }

    return created, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (user.User, string, error) {
    if email == "" || password == "" {
        return user.User{}, "", errors.New("email and password are required")
    }

    entity, err := s.repo.GetByEmail(ctx, email)
    if err != nil {
        return user.User{}, "", errors.New("invalid credentials")
    }

    if err := bcrypt.CompareHashAndPassword([]byte(entity.PasswordHash), []byte(password)); err != nil {
        return user.User{}, "", errors.New("invalid credentials")
    }

    token, err := s.generateToken(entity.ID)
    if err != nil {
        return user.User{}, "", err
    }

    return entity, token, nil
}

func (s *AuthService) generateToken(userID string) (string, error) {
    if len(s.jwtSecret) == 0 {
        return "", errors.New("missing jwt secret")
    }

    claims := jwt.MapClaims{
        "userId": userID,
        "exp":    time.Now().Add(s.tokenTTL).Unix(),
        "iat":    time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}