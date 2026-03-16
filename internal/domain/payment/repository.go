package payment

import "context"

// OrderRepository persists payment orders.
type OrderRepository interface {
	Create(ctx context.Context, order Order) (Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	GetByReferenceID(ctx context.Context, referenceID string) (Order, error)
	GetByUserID(ctx context.Context, userID string) ([]Order, error)
	GetPendingOrders(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, order Order) (Order, error)
}
