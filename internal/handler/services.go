package handler

import (
	"MyKitchen/internal/domain"
	"context"
)

type MenuItemService interface {
	GetMenuItemsByRestaurantID(context.Context, domain.RestaurantID) ([]domain.MenuItem, error)
	Add(context.Context, domain.MenuItem) (*domain.MenuItem, error)
	Update(context.Context, domain.MenuItemID, domain.RestaurantID, func(*domain.MenuItem) error) (*domain.MenuItem, error)
	Delete(context.Context, domain.MenuItemID, domain.RestaurantID) error
}

type OrderService interface {
	AddOrder(context.Context, domain.Order) (*domain.Order, error)
	GetOrdersByRestaurantID(context.Context, domain.RestaurantID) ([]domain.Order, error)
	ChangeStatusForRestaurant(context.Context, domain.RestaurantID, domain.OrderID, domain.OrderStatus) error
}

type RestaurantService interface {
	GetRestaurants(context.Context) ([]domain.Restaurant, error)
	GetRestaurantByID(context.Context, domain.RestaurantID) (*domain.Restaurant, error)
	RegisterRestaurant(context.Context, domain.Restaurant) (*domain.Restaurant, error)
	DeleteRestaurant(context.Context, domain.RestaurantID) (bool, error)
	UpdateAcceptingOrders(context.Context, domain.RestaurantID, bool) (bool, error)
}
