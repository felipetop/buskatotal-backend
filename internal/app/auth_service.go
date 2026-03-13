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

type AuthResult struct {
    User  user.User
    Token string
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (AuthResult, error) {
    if email == "" || password == "" {
        return AuthResult{}, errors.New("email and password are required")
    }

    if name == "" {
        name = email
    }

    if _, err := s.repo.GetByEmail(ctx, email); err == nil {
        return AuthResult{}, errors.New("email already registered")
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return AuthResult{}, errors.New("could not hash password")
    }

    created, err := s.repo.Create(ctx, user.User{
        Name:         name,
        Email:        email,
        PasswordHash: string(hash),
    })
    if err != nil {
        return AuthResult{}, err
    }

    token, err := s.generateToken(created.ID)
    if err != nil {
        return AuthResult{}, err
    }

    return AuthResult{User: created, Token: token}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
    if email == "" || password == "" {
        return AuthResult{}, errors.New("email and password are required")
    }

    entity, err := s.repo.GetByEmail(ctx, email)
    if err != nil {
        return AuthResult{}, errors.New("invalid credentials")
    }

    if err := bcrypt.CompareHashAndPassword([]byte(entity.PasswordHash), []byte(password)); err != nil {
        return AuthResult{}, errors.New("invalid credentials")
    }

    token, err := s.generateToken(entity.ID)
    if err != nil {
        return AuthResult{}, err
    }

    return AuthResult{User: entity, Token: token}, nil
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