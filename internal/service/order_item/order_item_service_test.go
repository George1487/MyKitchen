package order_item_test

import (
	"context"
	"errors"
	"testing"

	"MyKitchen/internal/domain"
	orderitemservice "MyKitchen/internal/service/order_item"
	"MyKitchen/tests/mocks"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAddOrderItem(t *testing.T) {
	item := domain.OrderItem{OrderID: 42, MenuItemID: 9, Quantity: 2, Price: 250}
	lookupErr := errors.New("order read failed")
	insertErr := errors.New("item insert failed")
	type in struct{ item domain.OrderItem }
	type out struct{ err error }
	addOrderItemTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockOrderItemRepository, *mocks.MockOrderRepository)
	}{
		{
			name: "adds item to existing order",
			in:   in{item: item},
			out:  out{},
			setup: func(items *mocks.MockOrderItemRepository, orders *mocks.MockOrderRepository) {
				gomock.InOrder(
					orders.EXPECT().GetByID(gomock.Any(), domain.OrderID(42)).Return(&domain.Order{ID: 42}, nil),
					items.EXPECT().AddOrderItem(gomock.Any(), item).Return(nil),
				)
			},
		},
		{
			name: "does not insert when order lookup fails",
			in:   in{item: item},
			setup: func(_ *mocks.MockOrderItemRepository, orders *mocks.MockOrderRepository) {
				orders.EXPECT().GetByID(gomock.Any(), domain.OrderID(42)).Return(nil, lookupErr)
			},
			out: out{err: lookupErr},
		},
		{
			name: "propagates insert failure",
			in:   in{item: item},
			setup: func(items *mocks.MockOrderItemRepository, orders *mocks.MockOrderRepository) {
				gomock.InOrder(
					orders.EXPECT().GetByID(gomock.Any(), domain.OrderID(42)).Return(&domain.Order{ID: 42}, nil),
					items.EXPECT().AddOrderItem(gomock.Any(), item).Return(insertErr),
				)
			},
			out: out{err: insertErr},
		},
	}

	for _, tt := range addOrderItemTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			items := mocks.NewMockOrderItemRepository(ctrl)
			orders := mocks.NewMockOrderRepository(ctrl)
			tt.setup(items, orders)
			err := orderitemservice.NewOrderItemService(items, orders).AddOrderItem(context.Background(), tt.in.item)
			require.ErrorIs(t, err, tt.out.err)
		})
	}
}

func TestGetItemsByOrderID(t *testing.T) {
	storageErr := errors.New("items read failed")
	type in struct {
		orderID   domain.OrderID
		repoItems []domain.OrderItem
		repoErr   error
	}
	type out struct {
		items []domain.OrderItem
		err   error
	}
	getItemsByOrderIDTest := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "returns items",
			in:   in{orderID: 42, repoItems: []domain.OrderItem{{ID: 1, OrderID: 42}, {ID: 2, OrderID: 42}}},
			out:  out{items: []domain.OrderItem{{ID: 1, OrderID: 42}, {ID: 2, OrderID: 42}}},
		},
		{name: "returns empty order", in: in{orderID: 42, repoItems: []domain.OrderItem{}}, out: out{items: []domain.OrderItem{}}},
		{name: "propagates read failure", in: in{orderID: 42, repoErr: storageErr}, out: out{err: storageErr}},
	}

	for _, tt := range getItemsByOrderIDTest {
		t.Run(tt.name, func(t *testing.T) {
			items := mocks.NewMockOrderItemRepository(gomock.NewController(t))
			items.EXPECT().GetByOrderID(gomock.Any(), tt.in.orderID).Return(tt.in.repoItems, tt.in.repoErr)
			got, err := orderitemservice.NewOrderItemService(items, nil).GetItemsByOrderID(context.Background(), tt.in.orderID)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.items, got)
		})
	}
}

func TestDeleteOrderItem(t *testing.T) {
	storageErr := errors.New("delete failed")
	type in struct {
		id      domain.OrderItemID
		repoErr error
	}
	type out struct{ err error }
	deleteOrderItemTest := []struct {
		name string
		in   in
		out  out
	}{
		{name: "deletes item", in: in{id: 7}, out: out{}},
		{name: "propagates delete failure", in: in{id: 7, repoErr: storageErr}, out: out{err: storageErr}},
	}
	for _, tt := range deleteOrderItemTest {
		t.Run(tt.name, func(t *testing.T) {
			items := mocks.NewMockOrderItemRepository(gomock.NewController(t))
			items.EXPECT().DeleteOrderItem(gomock.Any(), tt.in.id).Return(tt.in.repoErr)
			err := orderitemservice.NewOrderItemService(items, nil).DeleteOrderItem(context.Background(), tt.in.id)
			require.ErrorIs(t, err, tt.out.err)
		})
	}
}
