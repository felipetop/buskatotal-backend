package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/payment"
)

type OrderRepository struct {
	client *firestore.Client
}

func NewOrderRepository(client *firestore.Client) *OrderRepository {
	return &OrderRepository{client: client}
}

func (r *OrderRepository) Create(ctx context.Context, order payment.Order) (payment.Order, error) {
	now := time.Now()
	order.ID = uuid.NewString()
	order.CreatedAt = now
	order.UpdatedAt = now

	if _, err := r.client.Collection("orders").Doc(order.ID).Set(ctx, order); err != nil {
		return payment.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (payment.Order, error) {
	snap, err := r.client.Collection("orders").Doc(id).Get(ctx)
	if err != nil {
		return payment.Order{}, err
	}
	var order payment.Order
	if err := snap.DataTo(&order); err != nil {
		return payment.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) GetByReferenceID(ctx context.Context, referenceID string) (payment.Order, error) {
	snaps, err := r.client.Collection("orders").Where("ReferenceID", "==", referenceID).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return payment.Order{}, err
	}
	if len(snaps) == 0 {
		return payment.Order{}, errors.New("order not found")
	}
	var order payment.Order
	if err := snaps[0].DataTo(&order); err != nil {
		return payment.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) GetByUserID(ctx context.Context, userID string) ([]payment.Order, error) {
	snaps, err := r.client.Collection("orders").Where("UserID", "==", userID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	orders := make([]payment.Order, 0, len(snaps))
	for _, snap := range snaps {
		var order payment.Order
		if err := snap.DataTo(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepository) GetPendingOrders(ctx context.Context) ([]payment.Order, error) {
	snaps, err := r.client.Collection("orders").Where("Status", "==", string(payment.StatusPending)).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	orders := make([]payment.Order, 0, len(snaps))
	for _, snap := range snaps {
		var order payment.Order
		if err := snap.DataTo(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepository) Update(ctx context.Context, order payment.Order) (payment.Order, error) {
	order.UpdatedAt = time.Now()
	if _, err := r.client.Collection("orders").Doc(order.ID).Set(ctx, order); err != nil {
		return payment.Order{}, err
	}
	return order, nil
}
