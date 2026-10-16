package service

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type RestaurantService struct {
	restaurantRepo repository.RestaurantRepository
}

func NewRestaurantService(restaurantRepo repository.RestaurantRepository) *RestaurantService {

	return &RestaurantService{restaurantRepo: restaurantRepo}
}

func (s *RestaurantService) GetRestaurants(ctx context.Context) ([]domain.Restaurant, error) {

	restaurants, err := s.restaurantRepo.GetAll(ctx)

	if err != nil {
		return nil, fmt.Errorf("не удалось получить список ресторанов: %w", err)
	}

	return restaurants, nil
}

func (s *RestaurantService) GetRestaurantByID(ctx context.Context, id domain.RestaurantID) (*domain.Restaurant, error) {

	restaurant, err := s.restaurantRepo.GetByID(ctx, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrRestaurantNotFound, err)
		}

		return nil, fmt.Errorf("не удалось получить ресторан: %w", err)
	}
	return restaurant, nil
}

func (s *RestaurantService) RegisterRestaurant(ctx context.Context, restaurant domain.Restaurant) (*domain.Restaurant, error) {
	restaurantID, err := s.restaurantRepo.AddRestaurant(ctx, restaurant)
	if err != nil {
		return nil, fmt.Errorf("не удалось добавить ресторан: %w", err)
	}
	restaurant.ID = restaurantID
	restaurant.IsAcceptingOrders = true

	return &restaurant, nil
}

func (s *RestaurantService) DeleteRestaurant(ctx context.Context, id domain.RestaurantID) (bool, error) {
	deleted, err := s.restaurantRepo.DeleteRestaurant(ctx, id)
	if err != nil {
		return false, fmt.Errorf("не удалось удалить ресторан: %w", err)
	}

	return deleted, nil
}

func (s *RestaurantService) UpdateAcceptingOrders(
	ctx context.Context,
	id domain.RestaurantID,
	isAcceptingOrders bool,
) (bool, error) {
	updated, err := s.restaurantRepo.UpdateAcceptingOrders(ctx, id, isAcceptingOrders)
	if err != nil {
		return false, fmt.Errorf("не удалось изменить приём заказов: %w", err)
	}

	return updated, nil
}
