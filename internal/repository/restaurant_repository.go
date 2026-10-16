package repository

import (
	"MyKitchen/internal/domain"
	"context"
)

type RestaurantRepository interface {
	GetByID(ctx context.Context, id domain.RestaurantID) (*domain.Restaurant, error)

	GetAll(ctx context.Context) ([]domain.Restaurant, error)

	AddRestaurant(ctx context.Context, restaurant domain.Restaurant) (domain.RestaurantID, error)

	DeleteRestaurant(ctx context.Context, id domain.RestaurantID) (bool, error)

	UpdateAcceptingOrders(ctx context.Context, id domain.RestaurantID, isAcceptingOrders bool) (bool, error)
}
