package order

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"MyKitchen/internal/service"
	"MyKitchen/internal/transaction"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type OrderService struct {
	orderRepo      repository.OrderRepository
	menuItemRepo   repository.MenuItemRepository
	orderItemRepo  repository.OrderItemRepository
	restaurantRepo repository.RestaurantRepository
	txManager      transaction.TxManager
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	menuItemRepo repository.MenuItemRepository,
	orderItemRepo repository.OrderItemRepository,
	restaurantRepo repository.RestaurantRepository,
	txManager transaction.TxManager,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		menuItemRepo:   menuItemRepo,
		orderItemRepo:  orderItemRepo,
		restaurantRepo: restaurantRepo,
		txManager:      txManager,
	}
}

func (s *OrderService) GetByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {

	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", service.ErrOrderNotFound, err)
		}

		return nil, fmt.Errorf("не удалось найти заказ: %w", err)
	}
	orderItems, err := s.orderItemRepo.GetByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить позиции заказа: %w", err)
	}
	order.OrderItems = orderItems

	return order, nil
}

func (s *OrderService) GetOrdersByRestaurantID(
	ctx context.Context,
	restaurantID domain.RestaurantID,
) ([]domain.Order, error) {
	orders, err := s.orderRepo.GetOrdersByRestaurantID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить заказы ресторана: %w", err)
	}
	for i := range orders {
		orderItems, err := s.orderItemRepo.GetByOrderID(ctx, orders[i].ID)
		if err != nil {
			return nil, fmt.Errorf("не удалось получить позиции заказа %d: %w", orders[i].ID, err)
		}
		orders[i].OrderItems = orderItems
	}

	return orders, nil
}

func (s *OrderService) AddOrder(ctx context.Context, order domain.Order) (*domain.Order, error) {
	var created *domain.Order
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.addOrder(txCtx, order)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, err
}

func (s *OrderService) addOrder(ctx context.Context, order domain.Order) (*domain.Order, error) {
	order.Address = strings.TrimSpace(order.Address)
	if order.Address == "" {
		return nil, fmt.Errorf("адрес доставки не может быть пустым")
	}

	if order.Status != domain.OrderStatusNew {
		return nil, fmt.Errorf("новый заказ должен иметь статус %q", domain.OrderStatusNew)
	}
	if len(order.OrderItems) == 0 {
		return nil, fmt.Errorf("заказ должен содержать хотя бы одну позицию")
	}

	restaurant, err := s.restaurantRepo.GetByIDForUpdate(ctx, order.RestaurantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", service.ErrRestaurantNotFound, err)
		}

		return nil, fmt.Errorf("не удалось получить ресторан: %w", err)
	}
	if !restaurant.IsAcceptingOrders {
		return nil, fmt.Errorf("ресторан не принимает заказы")
	}

	requestedQuantities := make(map[domain.MenuItemID]int64)
	for _, orderItem := range order.OrderItems {
		if orderItem.Quantity <= 0 {
			return nil, fmt.Errorf("количество позиции %d должно быть положительным", orderItem.MenuItemID)
		}

		requestedQuantities[orderItem.MenuItemID] += orderItem.Quantity
	}

	menuItemIDs := make([]domain.MenuItemID, 0, len(requestedQuantities))
	for menuItemID := range requestedQuantities {
		menuItemIDs = append(menuItemIDs, menuItemID)
	}
	sort.Slice(menuItemIDs, func(i, j int) bool { return menuItemIDs[i] < menuItemIDs[j] })
	menuItems := make(map[domain.MenuItemID]*domain.MenuItem, len(requestedQuantities))
	for _, menuItemID := range menuItemIDs {
		requestedQuantity := requestedQuantities[menuItemID]
		menuItem, err := s.menuItemRepo.GetByIDForUpdate(ctx, menuItemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: %w", service.ErrMenuItemNotFound, err)
			}

			return nil, fmt.Errorf("не удалось получить блюдо %d: %w", menuItemID, err)
		}
		if menuItem.RestaurantID != order.RestaurantID {
			return nil, fmt.Errorf("блюдо %d не принадлежит ресторану %d", menuItemID, order.RestaurantID)
		}
		if !menuItem.IsAvailable {
			return nil, fmt.Errorf("блюдо %d недоступно для заказа", menuItemID)
		}
		if menuItem.AvailableQuantity < requestedQuantity {
			return nil, fmt.Errorf("недостаточно блюда %d: доступно %d, запрошено %d",
				menuItemID, menuItem.AvailableQuantity, requestedQuantity)
		}

		menuItems[menuItemID] = menuItem
	}

	order.TotalPrice = 0
	for i := range order.OrderItems {
		menuItem := menuItems[order.OrderItems[i].MenuItemID]
		order.OrderItems[i].Price = menuItem.Price
		order.TotalPrice += menuItem.Price * order.OrderItems[i].Quantity
	}

	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}

	orderID, err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("не удалось добавить заказ: %w", err)
	}
	order.ID = orderID

	return &order, nil
}

func (s *OrderService) AcceptOrder(ctx context.Context, id domain.OrderID) error {
	return s.ChangeStatus(ctx, id, domain.OrderStatusAccepted)
}

func (s *OrderService) CancelOrder(ctx context.Context, id domain.OrderID) error {
	return s.ChangeStatus(ctx, id, domain.OrderStatusCancelled)
}

func (s *OrderService) DeliverOrder(ctx context.Context, id domain.OrderID) error {
	return s.ChangeStatus(ctx, id, domain.OrderStatusDelivered)
}

func (s *OrderService) ChangeStatus(ctx context.Context, id domain.OrderID, newStatus domain.OrderStatus) error {
	if !isValidOrderStatus(newStatus) {
		return fmt.Errorf("некорректный статус заказа: %q", newStatus)
	}
	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		order, err := s.orderRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %w", service.ErrOrderNotFound, err)
			}

			return fmt.Errorf("не удалось найти заказ: %w", err)
		}

		return s.changeStatus(txCtx, order, newStatus)
	})
}

func (s *OrderService) ChangeStatusForRestaurant(
	ctx context.Context,
	restaurantID domain.RestaurantID,
	orderID domain.OrderID,
	newStatus domain.OrderStatus,
) error {
	if !isValidOrderStatus(newStatus) {
		return fmt.Errorf("некорректный статус заказа: %q", newStatus)
	}
	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		order, err := s.orderRepo.GetByIDForUpdate(txCtx, orderID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %w", service.ErrOrderNotFound, err)
			}

			return fmt.Errorf("не удалось найти заказ: %w", err)
		}
		if order.RestaurantID != restaurantID {
			return fmt.Errorf("заказ %d не принадлежит ресторану %d", orderID, restaurantID)
		}

		return s.changeStatus(txCtx, order, newStatus)
	})
}

func (s *OrderService) changeStatus(ctx context.Context, order *domain.Order, newStatus domain.OrderStatus) error {
	if !isValidOrderStatus(newStatus) {
		return fmt.Errorf("некорректный статус заказа: %q", newStatus)
	}
	if !isValidStatusTransition(order.Status, newStatus) {
		return fmt.Errorf("недопустимый переход статуса заказа: %s -> %s", order.Status, newStatus)
	}

	var (
		updated bool
		err     error
	)
	if newStatus == domain.OrderStatusCancelled {
		updated, err = s.orderRepo.CancelOrder(ctx, order.ID, order.Status)
	} else {
		updated, err = s.orderRepo.UpdateStatus(ctx, order.ID, order.Status, newStatus)
	}
	if err != nil {
		return fmt.Errorf("не удалось изменить статус заказа: %w", err)
	}
	if !updated {
		return fmt.Errorf("статус заказа уже был изменён")
	}

	return nil
}

func isValidOrderStatus(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusNew,
		domain.OrderStatusAccepted,
		domain.OrderStatusCancelled,
		domain.OrderStatusDelivered:
		return true
	default:
		return false
	}
}

func isValidStatusTransition(fromStatus, toStatus domain.OrderStatus) bool {
	switch fromStatus {
	case domain.OrderStatusNew:
		return toStatus == domain.OrderStatusAccepted || toStatus == domain.OrderStatusCancelled
	case domain.OrderStatusAccepted:
		return toStatus == domain.OrderStatusDelivered || toStatus == domain.OrderStatusCancelled
	default:
		return false
	}
}

func (s *OrderService) DeleteOrder(ctx context.Context, id domain.OrderID) error {
	return s.txManager.Do(ctx, func(txCtx context.Context) error {
		order, err := s.orderRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %w", service.ErrOrderNotFound, err)
			}

			return fmt.Errorf("не удалось найти заказ: %w", err)
		}

		err = s.orderRepo.DeleteOrder(txCtx, order.ID)
		if err != nil {
			return fmt.Errorf("не удалось удалить заказ: %w", err)
		}
		return nil
	})
}
