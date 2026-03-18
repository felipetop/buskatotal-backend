package app

import (
    "context"
    "errors"
    "regexp"
    "time"

    jwt "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"

    "buskatotal-backend/internal/domain/user"
)

func validatePassword(password string) error {
    if len(password) < 10 {
        return errors.New("password must be at least 10 characters long")
    }
    if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
        return errors.New("password must contain at least one uppercase letter (A–Z)")
    }
    if !regexp.MustCompile(`[a-z]`).MatchString(password) {
        return errors.New("password must contain at least one lowercase letter (a–z)")
    }
    if !regexp.MustCompile(`[0-9]`).MatchString(password) {
        return errors.New("password must contain at least one number (0–9)")
    }
    if !regexp.MustCompile(`[@!#$%]`).MatchString(password) {
        return errors.New("password must contain at least one special character (@, !, #, $, %)")
    }
    return nil
}

// adminEmails is the list of emails that automatically get admin role.
var adminEmails = map[string]bool{
    "dcparticular2014@gmail.com": true,
    "felipejorgetop@gmail.com":   true,
}

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

    if err := validatePassword(password); err != nil {
        return user.User{}, "", err
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

    role := user.RoleUser
    if adminEmails[email] {
        role = user.RoleAdmin
    }

    created, err := s.repo.Create(ctx, user.User{
        Name:         name,
        Email:        email,
        Role:         role,
        PasswordHash: string(hash),
    })
    if err != nil {
        return user.User{}, "", err
    }

    token, err := s.generateToken(created.ID, created.Role)
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

    // Upgrade role if email is in admin list but user was created before admin feature
    if adminEmails[entity.Email] && entity.Role != user.RoleAdmin {
        entity.Role = user.RoleAdmin
        s.repo.Update(ctx, entity)
    }

    token, err := s.generateToken(entity.ID, entity.Role)
    if err != nil {
        return user.User{}, "", err
    }

    return entity, token, nil
}

func (s *AuthService) generateToken(userID, role string) (string, error) {
    if len(s.jwtSecret) == 0 {
        return "", errors.New("missing jwt secret")
    }

    if role == "" {
        role = user.RoleUser
    }

    claims := jwt.MapClaims{
        "userId": userID,
        "role":   role,
        "exp":    time.Now().Add(s.tokenTTL).Unix(),
        "iat":    time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}