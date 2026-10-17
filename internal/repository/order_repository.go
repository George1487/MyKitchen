package repository

import (
	"MyKitchen/internal/domain"
	"context"
)

type OrderRepository interface {
	GetByID(ctx context.Context, orderID domain.OrderID) (*domain.Order, error)
	GetByIDForUpdate(ctx context.Context, orderID domain.OrderID) (*domain.Order, error)

	GetOrdersByRestaurantID(ctx context.Context, restaurantID domain.RestaurantID) ([]domain.Order, error)

	CreateOrder(ctx context.Context, order domain.Order) (domain.OrderID, error)

	CancelOrder(
		ctx context.Context,
		orderID domain.OrderID,
		fromStatus domain.OrderStatus,
	) (bool, error)

	UpdateStatus(
		ctx context.Context,
		orderID domain.OrderID,
		fromStatus domain.OrderStatus,
		toStatus domain.OrderStatus,
	) (bool, error)

	DeleteOrder(ctx context.Context, orderID domain.OrderID) error
}
