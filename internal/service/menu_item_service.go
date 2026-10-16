package service

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type MenuItemService struct {
	menuItemRepo   repository.MenuItemRepository
	restaurantRepo repository.RestaurantRepository
}

func NewMenuItemService(menuItemRepo repository.MenuItemRepository,
	restaurantRepo repository.RestaurantRepository) *MenuItemService {
	return &MenuItemService{menuItemRepo: menuItemRepo, restaurantRepo: restaurantRepo}
}

func (s *MenuItemService) GetById(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error) {

	menuItem, err := s.menuItemRepo.GetByID(ctx, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrMenuItemNotFound, err)
		}

		return nil, fmt.Errorf("не удалось получить товар: %w", err)
	}

	return menuItem, nil
}

func (s *MenuItemService) GetMenuItemsByRestaurantID(
	ctx context.Context,
	restaurantID domain.RestaurantID,
) ([]domain.MenuItem, error) {
	_, err := s.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrRestaurantNotFound, err)
		}

		return nil, fmt.Errorf("не удалось найти ресторан: %w", err)
	}

	menuItems, err := s.menuItemRepo.GetByRestaurantID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить меню: %w", err)
	}
	return menuItems, nil
}

func (s *MenuItemService) Add(ctx context.Context, item domain.MenuItem) (*domain.MenuItem, error) {
	_, err := s.restaurantRepo.GetByID(ctx, item.RestaurantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrRestaurantNotFound, err)
		}

		return nil, fmt.Errorf("не удалось найти ресторан: %w", err)
	}

	menuItemID, err := s.menuItemRepo.Add(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("не удалось добавить позицию: %w", err)
	}
	item.ID = menuItemID

	return &item, nil
}

func (s *MenuItemService) Delete(ctx context.Context, id domain.MenuItemID) error {

	err := s.menuItemRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("не удалось удалить позицию: %w", err)
	}
	return nil
}

func (s *MenuItemService) Update(ctx context.Context, id domain.MenuItemID, item domain.MenuItem) error {

	err := s.menuItemRepo.Update(ctx, id, item)
	if err != nil {
		return fmt.Errorf("не удалось обновить позицию: %w", err)
	}
	return nil
}
