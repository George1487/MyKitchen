package order_test

import (
	"MyKitchen/internal/domain"
	service "MyKitchen/internal/service/order"
	"MyKitchen/tests/mocks"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type passthroughTxManager struct{}

func (passthroughTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func requireErrorText(t *testing.T, err error, want string) {
	t.Helper()
	switch want {
	case "":
		require.NoError(t, err)
	default:
		require.ErrorContains(t, err, want)
	}
}

func TestAddOrder(t *testing.T) {
	createdAt := time.Date(2026, time.October, 17, 12, 0, 0, 0, time.UTC)
	type in struct{ order domain.Order }
	type out struct {
		order *domain.Order
		err   string
	}
	addOrderTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemRepository, *mocks.MockOrderRepository, *mocks.MockRestaurantRepository)
	}{
		{
			name: "creates order and calculates price for duplicate menu items",
			in: in{order: domain.Order{
				ClientID:     42,
				RestaurantID: 17,
				Address:      "  Москва, ул. Тверская, 1  ",
				Status:       domain.OrderStatusNew,
				CreatedAt:    createdAt,
				OrderItems: []domain.OrderItem{
					{MenuItemID: 5, Quantity: 2},
					{MenuItemID: 5, Quantity: 1},
				},
			}},
			out: out{order: &domain.Order{
				ID: 11, ClientID: 42, RestaurantID: 17, Address: "Москва, ул. Тверская, 1",
				TotalPrice: 750, Status: domain.OrderStatusNew, CreatedAt: createdAt,
				OrderItems: []domain.OrderItem{{MenuItemID: 5, Price: 250, Quantity: 2}, {MenuItemID: 5, Price: 250, Quantity: 1}},
			}},
			setup: func(
				menuItemRepo *mocks.MockMenuItemRepository,
				orderRepo *mocks.MockOrderRepository,
				restaurantRepo *mocks.MockRestaurantRepository,
			) {
				restaurantRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{
					ID:                17,
					IsAcceptingOrders: true,
				}, nil)
				menuItemRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(5)).Return(&domain.MenuItem{
					ID:                5,
					RestaurantID:      17,
					Price:             250,
					IsAvailable:       true,
					AvailableQuantity: 3,
				}, nil)
				orderRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, got domain.Order) (domain.OrderID, error) {
						require.Equal(t, "Москва, ул. Тверская, 1", got.Address)
						require.Equal(t, int64(750), got.TotalPrice)
						for _, item := range got.OrderItems {
							require.Equal(t, int64(250), item.Price)
						}
						return 11, nil
					},
				)
			},
		},
		{
			name: "rejects empty delivery address",
			in: in{order: domain.Order{
				RestaurantID: 17,
				Address:      "   ",
				Status:       domain.OrderStatusNew,
				OrderItems:   []domain.OrderItem{{MenuItemID: 5, Quantity: 1}},
			}},
			setup: func(*mocks.MockMenuItemRepository, *mocks.MockOrderRepository, *mocks.MockRestaurantRepository) {},
			out:   out{err: "адрес доставки не может быть пустым"},
		},
		{
			name: "rejects unavailable menu item",
			in: in{order: domain.Order{
				RestaurantID: 17,
				Address:      "Москва, ул. Тверская, 1",
				Status:       domain.OrderStatusNew,
				OrderItems:   []domain.OrderItem{{MenuItemID: 5, Quantity: 1}},
			}},
			setup: func(
				menuItemRepo *mocks.MockMenuItemRepository,
				_ *mocks.MockOrderRepository,
				restaurantRepo *mocks.MockRestaurantRepository,
			) {
				restaurantRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{
					ID:                17,
					IsAcceptingOrders: true,
				}, nil)
				menuItemRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(5)).Return(&domain.MenuItem{
					ID:                5,
					RestaurantID:      17,
					Price:             250,
					IsAvailable:       false,
					AvailableQuantity: 10,
				}, nil)
			},
			out: out{err: "недоступно для заказа"},
		},
	}

	for _, tt := range addOrderTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			menuItemRepo := mocks.NewMockMenuItemRepository(ctrl)
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			restaurantRepo := mocks.NewMockRestaurantRepository(ctrl)
			tt.setup(menuItemRepo, orderRepo, restaurantRepo)

			service := service.NewOrderService(
				orderRepo,
				menuItemRepo,
				mocks.NewMockOrderItemRepository(ctrl),
				restaurantRepo,
				passthroughTxManager{},
			)
			createdOrder, err := service.AddOrder(context.Background(), tt.in.order)
			requireErrorText(t, err, tt.out.err)
			require.Equal(t, tt.out.order, createdOrder)
		})
	}
}

func TestChangeStatus(t *testing.T) {
	type in struct {
		id   domain.OrderID
		next domain.OrderStatus
	}
	type out struct{ err string }
	changeStatusTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockOrderRepository)
	}{
		{
			name: "restaurant accepts new order",
			in:   in{id: 10, next: domain.OrderStatusAccepted},
			out:  out{},
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusNew}, nil)
				orderRepo.EXPECT().UpdateStatus(
					gomock.Any(),
					domain.OrderID(10),
					domain.OrderStatusNew,
					domain.OrderStatusAccepted,
				).Return(true, nil)
			},
		},
		{
			name: "restaurant cancels accepted order",
			in:   in{id: 10, next: domain.OrderStatusCancelled},
			out:  out{},
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusAccepted}, nil)
				orderRepo.EXPECT().CancelOrder(
					gomock.Any(),
					domain.OrderID(10),
					domain.OrderStatusAccepted,
				).Return(true, nil)
			},
		},
		{
			name: "cannot deliver new order before acceptance",
			in:   in{id: 10, next: domain.OrderStatusDelivered},
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByIDForUpdate(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusNew}, nil)
			},
			out: out{err: "недопустимый переход"},
		},
	}

	for _, tt := range changeStatusTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			tt.setup(orderRepo)

			service := service.NewOrderService(
				orderRepo,
				mocks.NewMockMenuItemRepository(ctrl),
				mocks.NewMockOrderItemRepository(ctrl),
				mocks.NewMockRestaurantRepository(ctrl),
				passthroughTxManager{},
			)
			err := service.ChangeStatus(context.Background(), tt.in.id, tt.in.next)
			requireErrorText(t, err, tt.out.err)
		})
	}
}
