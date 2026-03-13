package user

import "time"

type User struct {
    ID        string    `json:"id" firestore:"id"`
    Name      string    `json:"name" firestore:"name"`
    Email     string    `json:"email" firestore:"email"`
    Balance   int64     `json:"balance" firestore:"balance"`
    CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}