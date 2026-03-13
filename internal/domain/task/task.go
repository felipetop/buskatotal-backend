package task

import "time"

type Task struct {
    ID          string    `json:"id" firestore:"id"`
    UserID      string    `json:"userId" firestore:"userId"`
    Title       string    `json:"title" firestore:"title"`
    Description string    `json:"description" firestore:"description"`
    Done        bool      `json:"done" firestore:"done"`
    CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt" firestore:"updatedAt"`
}