package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/payment"
)

type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]payment.Order // key: ID
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{orders: make(map[string]payment.Order)}
}

func (r *OrderRepository) Create(ctx context.Context, order payment.Order) (payment.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order.ID = uuid.NewString()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	r.orders[order.ID] = order
	return order, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (payment.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	o, ok := r.orders[id]
	if !ok {
		return payment.Order{}, errors.New("order not found")
	}
	return o, nil
}

func (r *OrderRepository) GetByReferenceID(ctx context.Context, referenceID string) (payment.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, o := range r.orders {
		if o.ReferenceID == referenceID {
			return o, nil
		}
	}
	return payment.Order{}, errors.New("order not found")
}

func (r *OrderRepository) GetByUserID(ctx context.Context, userID string) ([]payment.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []payment.Order
	for _, o := range r.orders {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (r *OrderRepository) GetPendingOrders(ctx context.Context) ([]payment.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []payment.Order
	for _, o := range r.orders {
		if o.Status == payment.StatusPending {
			result = append(result, o)
		}
	}
	return result, nil
}

func (r *OrderRepository) Update(ctx context.Context, order payment.Order) (payment.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.orders[order.ID]; !ok {
		return payment.Order{}, errors.New("order not found")
	}
	order.UpdatedAt = time.Now()
	r.orders[order.ID] = order
	return order, nil
}
