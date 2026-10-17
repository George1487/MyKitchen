package order_item

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"context"
	"fmt"
)

type OrderItemService struct {
	orderItemRepo repository.OrderItemRepository

	orderRepo repository.OrderRepository
}

func NewOrderItemService(orderItemRepo repository.OrderItemRepository,
	orderRepo repository.OrderRepository) *OrderItemService {
	return &OrderItemService{orderItemRepo: orderItemRepo, orderRepo: orderRepo}
}

func (s *OrderItemService) GetById(ctx context.Context, id domain.OrderItemID) (*domain.OrderItem, error) {

	orderItem, err := s.orderItemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить товар заказа: %w", err)
	}

	return orderItem, nil
}

func (s *OrderItemService) GetItemsByOrderID(ctx context.Context, id domain.OrderID) ([]domain.OrderItem, error) {

	orderItems, err := s.orderItemRepo.GetByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить товары заказа: %w", err)
	}
	return orderItems, nil
}

func (s *OrderItemService) AddOrderItem(ctx context.Context, orderItem domain.OrderItem) error {

	_, err := s.orderRepo.GetByID(ctx, orderItem.OrderID)

	if err != nil {
		return fmt.Errorf("не удалось найти заказ %w", err)
	}

	err = s.orderItemRepo.AddOrderItem(ctx, orderItem)
	if err != nil {
		return fmt.Errorf("не удалось добавить товар заказа: %w", err)
	}

	return nil
}

func (s *OrderItemService) DeleteOrderItem(ctx context.Context, orderItemID domain.OrderItemID) error {

	err := s.orderItemRepo.DeleteOrderItem(ctx, orderItemID)
	if err != nil {
		return fmt.Errorf("не удалось удалить товар заказа: %w", err)
	}

	return nil
}
