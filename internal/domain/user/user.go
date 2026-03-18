package user

import "time"

const (
    RoleUser  = "user"
    RoleAdmin = "admin"
)

type User struct {
    ID        string    `json:"id" firestore:"id"`
    Name      string    `json:"name" firestore:"name"`
    Email     string    `json:"email" firestore:"email"`
    Role      string    `json:"role" firestore:"role"`
    PasswordHash string `json:"-" firestore:"passwordHash"`
    Balance   int64     `json:"balance" firestore:"balance"`
    CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

func (u User) IsAdmin() bool {
    return u.Role == RoleAdmin
}