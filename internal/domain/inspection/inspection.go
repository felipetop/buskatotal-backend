package inspection

import "time"

type Inspection struct {
	ID        string    `json:"id" firestore:"id"`
	UserID    string    `json:"user_id" firestore:"userID"`
	Protocol  string    `json:"protocol" firestore:"protocol"`
	Customer  string    `json:"customer" firestore:"customer"`
	Cellphone string    `json:"cellphone" firestore:"cellphone"`
	Plate     string    `json:"plate,omitempty" firestore:"plate"`
	Chassis   string    `json:"chassis,omitempty" firestore:"chassis"`
	Notes     string    `json:"notes,omitempty" firestore:"notes"`
	Status    string    `json:"status" firestore:"status"`
	CreatedAt time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updatedAt"`
}
