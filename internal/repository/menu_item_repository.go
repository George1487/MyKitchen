package repository

import (
	"MyKitchen/internal/domain"
	"context"
)

type MenuItemRepository interface {
	GetByID(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error)
	GetByIDForUpdate(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error)

	GetByRestaurantID(ctx context.Context, restaurant domain.RestaurantID) ([]domain.MenuItem, error)

	Add(ctx context.Context, item domain.MenuItem) (domain.MenuItemID, error)

	Delete(ctx context.Context, id domain.MenuItemID) error

	Update(ctx context.Context, id domain.MenuItemID, item domain.MenuItem) error
}
