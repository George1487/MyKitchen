package service_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/service"
	"MyKitchen/tests/mocks"
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestOrderServiceAddOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   domain.Order
		setup   func(*mocks.MockMenuItemRepository, *mocks.MockOrderRepository, *mocks.MockRestaurantRepository)
		wantErr string
	}{
		{
			name: "creates order and calculates price for duplicate menu items",
			order: domain.Order{
				ClientID:     42,
				RestaurantID: 17,
				Address:      "  Москва, ул. Тверская, 1  ",
				Status:       domain.OrderStatusNew,
				OrderItems: []domain.OrderItem{
					{MenuItemID: 5, Quantity: 2},
					{MenuItemID: 5, Quantity: 1},
				},
			},
			setup: func(
				menuItemRepo *mocks.MockMenuItemRepository,
				orderRepo *mocks.MockOrderRepository,
				restaurantRepo *mocks.MockRestaurantRepository,
			) {
				restaurantRepo.EXPECT().GetByID(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{
					ID:                17,
					IsAcceptingOrders: true,
				}, nil)
				menuItemRepo.EXPECT().GetByID(gomock.Any(), domain.MenuItemID(5)).Return(&domain.MenuItem{
					ID:                5,
					RestaurantID:      17,
					Price:             250,
					IsAvailable:       true,
					AvailableQuantity: 3,
				}, nil)
				orderRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, got domain.Order) (domain.OrderID, error) {
						if got.Address != "Москва, ул. Тверская, 1" {
							t.Errorf("address = %q, want trimmed address", got.Address)
						}
						if got.TotalPrice != 750 {
							t.Errorf("total price = %d, want 750", got.TotalPrice)
						}
						for i, item := range got.OrderItems {
							if item.Price != 250 {
								t.Errorf("item %d price = %d, want 250", i, item.Price)
							}
						}
						return 11, nil
					},
				)
			},
		},
		{
			name: "rejects empty delivery address",
			order: domain.Order{
				RestaurantID: 17,
				Address:      "   ",
				Status:       domain.OrderStatusNew,
				OrderItems:   []domain.OrderItem{{MenuItemID: 5, Quantity: 1}},
			},
			setup:   func(*mocks.MockMenuItemRepository, *mocks.MockOrderRepository, *mocks.MockRestaurantRepository) {},
			wantErr: "адрес доставки не может быть пустым",
		},
		{
			name: "rejects unavailable menu item",
			order: domain.Order{
				RestaurantID: 17,
				Address:      "Москва, ул. Тверская, 1",
				Status:       domain.OrderStatusNew,
				OrderItems:   []domain.OrderItem{{MenuItemID: 5, Quantity: 1}},
			},
			setup: func(
				menuItemRepo *mocks.MockMenuItemRepository,
				_ *mocks.MockOrderRepository,
				restaurantRepo *mocks.MockRestaurantRepository,
			) {
				restaurantRepo.EXPECT().GetByID(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{
					ID:                17,
					IsAcceptingOrders: true,
				}, nil)
				menuItemRepo.EXPECT().GetByID(gomock.Any(), domain.MenuItemID(5)).Return(&domain.MenuItem{
					ID:                5,
					RestaurantID:      17,
					Price:             250,
					IsAvailable:       false,
					AvailableQuantity: 10,
				}, nil)
			},
			wantErr: "недоступно для заказа",
		},
	}

	for _, tt := range tests {
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
			)
			createdOrder, err := service.AddOrder(context.Background(), tt.order)

			if tt.wantErr == "" && err != nil {
				t.Fatalf("AddOrder() error = %v", err)
			}
			if tt.wantErr == "" && (createdOrder == nil || createdOrder.ID != 11) {
				t.Fatalf("AddOrder() ID = %v, want 11", createdOrder)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("AddOrder() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestOrderServiceChangeStatus(t *testing.T) {
	tests := []struct {
		name    string
		current domain.OrderStatus
		next    domain.OrderStatus
		setup   func(*mocks.MockOrderRepository)
		wantErr string
	}{
		{
			name:    "restaurant accepts new order",
			current: domain.OrderStatusNew,
			next:    domain.OrderStatusAccepted,
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByID(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusNew}, nil)
				orderRepo.EXPECT().UpdateStatus(
					gomock.Any(),
					domain.OrderID(10),
					domain.OrderStatusNew,
					domain.OrderStatusAccepted,
				).Return(true, nil)
			},
		},
		{
			name:    "restaurant cancels accepted order",
			current: domain.OrderStatusAccepted,
			next:    domain.OrderStatusCancelled,
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByID(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusAccepted}, nil)
				orderRepo.EXPECT().CancelOrder(
					gomock.Any(),
					domain.OrderID(10),
					domain.OrderStatusAccepted,
				).Return(true, nil)
			},
		},
		{
			name:    "cannot deliver new order before acceptance",
			current: domain.OrderStatusNew,
			next:    domain.OrderStatusDelivered,
			setup: func(orderRepo *mocks.MockOrderRepository) {
				orderRepo.EXPECT().GetByID(gomock.Any(), domain.OrderID(10)).Return(&domain.Order{ID: 10, Status: domain.OrderStatusNew}, nil)
			},
			wantErr: "недопустимый переход",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			tt.setup(orderRepo)

			service := service.NewOrderService(
				orderRepo,
				mocks.NewMockMenuItemRepository(ctrl),
				mocks.NewMockOrderItemRepository(ctrl),
				mocks.NewMockRestaurantRepository(ctrl),
			)
			err := service.ChangeStatus(context.Background(), 10, tt.next)

			if tt.wantErr == "" && err != nil {
				t.Fatalf("ChangeStatus() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ChangeStatus() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
