package menu_item

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"MyKitchen/internal/service"
	"MyKitchen/internal/transaction"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type MenuItemService struct {
	menuItemRepo   repository.MenuItemRepository
	restaurantRepo repository.RestaurantRepository
	txManager      transaction.TxManager
}

func NewMenuItemService(menuItemRepo repository.MenuItemRepository,
	restaurantRepo repository.RestaurantRepository, txManager transaction.TxManager) *MenuItemService {
	return &MenuItemService{menuItemRepo: menuItemRepo, restaurantRepo: restaurantRepo, txManager: txManager}
}

func (s *MenuItemService) GetById(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error) {

	menuItem, err := s.menuItemRepo.GetByID(ctx, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", service.ErrMenuItemNotFound, err)
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
			return nil, fmt.Errorf("%w: %w", service.ErrRestaurantNotFound, err)
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
	var created *domain.MenuItem
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.add(txCtx, item)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, err
}

func (s *MenuItemService) add(ctx context.Context, item domain.MenuItem) (*domain.MenuItem, error) {
	_, err := s.restaurantRepo.GetByIDForUpdate(ctx, item.RestaurantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", service.ErrRestaurantNotFound, err)
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

func (s *MenuItemService) Delete(ctx context.Context, id domain.MenuItemID, restaurantID domain.RestaurantID) error {
	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		item, err := s.menuItemRepo.GetByIDForUpdate(txCtx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ErrMenuItemNotFound
		}
		if err != nil {
			return fmt.Errorf("не удалось получить позицию: %w", err)
		}
		if item.RestaurantID != restaurantID {
			return service.ErrMenuItemNotFound
		}
		if err := s.menuItemRepo.Delete(txCtx, id); err != nil {
			return fmt.Errorf("не удалось удалить позицию: %w", err)
		}
		return nil
	})
}

func (s *MenuItemService) Update(ctx context.Context, id domain.MenuItemID, restaurantID domain.RestaurantID, apply func(*domain.MenuItem) error) (*domain.MenuItem, error) {
	var updated *domain.MenuItem
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		item, err := s.menuItemRepo.GetByIDForUpdate(txCtx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ErrMenuItemNotFound
		}
		if err != nil {
			return fmt.Errorf("не удалось получить позицию: %w", err)
		}
		if item.RestaurantID != restaurantID {
			return service.ErrMenuItemNotFound
		}
		if err := apply(item); err != nil {
			return err
		}
		if err := s.menuItemRepo.Update(txCtx, id, *item); err != nil {
			return fmt.Errorf("не удалось обновить позицию: %w", err)
		}
		updated = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, err
}
