package repository

import (
	"MyKitchen/internal/domain"
	"context"
)

type OrderItemRepository interface {
	GetByID(ctx context.Context, orderItemID domain.OrderItemID) (*domain.OrderItem, error)

	GetByOrderID(ctx context.Context, orderID domain.OrderID) ([]domain.OrderItem, error)

	AddOrderItem(ctx context.Context, orderItem domain.OrderItem) error

	DeleteOrderItem(ctx context.Context, orderItemID domain.OrderItemID) error
}
