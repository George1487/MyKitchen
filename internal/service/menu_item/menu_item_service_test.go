package menu_item_test

import (
	"context"
	"errors"
	"testing"

	"MyKitchen/internal/domain"
	"MyKitchen/internal/service"
	menuservice "MyKitchen/internal/service/menu_item"
	"MyKitchen/tests/mocks"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type passthroughTxManager struct{}

func (passthroughTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestAdd(t *testing.T) {
	item := domain.MenuItem{Name: "Soup", RestaurantID: 17, Price: 250, AvailableQuantity: 3}
	created := item
	created.ID = 9
	storageErr := errors.New("insert failed")
	type in struct{ item domain.MenuItem }
	type out struct {
		item *domain.MenuItem
		err  error
	}
	addTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemRepository, *mocks.MockRestaurantRepository)
	}{
		{
			name: "creates item for existing restaurant",
			in:   in{item: item},
			setup: func(items *mocks.MockMenuItemRepository, restaurants *mocks.MockRestaurantRepository) {
				gomock.InOrder(
					restaurants.EXPECT().GetByIDForUpdate(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{ID: 17}, nil),
					items.EXPECT().Add(gomock.Any(), item).Return(domain.MenuItemID(9), nil),
				)
			},
			out: out{item: &created},
		},
		{
			name: "rejects missing restaurant",
			in:   in{item: item},
			setup: func(_ *mocks.MockMenuItemRepository, restaurants *mocks.MockRestaurantRepository) {
				restaurants.EXPECT().GetByIDForUpdate(gomock.Any(), domain.RestaurantID(17)).Return(nil, pgx.ErrNoRows)
			},
			out: out{err: service.ErrRestaurantNotFound},
		},
		{
			name: "propagates insert failure",
			in:   in{item: item},
			setup: func(items *mocks.MockMenuItemRepository, restaurants *mocks.MockRestaurantRepository) {
				restaurants.EXPECT().GetByIDForUpdate(gomock.Any(), domain.RestaurantID(17)).Return(&domain.Restaurant{ID: 17}, nil)
				items.EXPECT().Add(gomock.Any(), item).Return(domain.MenuItemID(0), storageErr)
			},
			out: out{err: storageErr},
		},
	}

	for _, tt := range addTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			items := mocks.NewMockMenuItemRepository(ctrl)
			restaurants := mocks.NewMockRestaurantRepository(ctrl)
			tt.setup(items, restaurants)
			got, err := menuservice.NewMenuItemService(items, restaurants, passthroughTxManager{}).Add(context.Background(), tt.in.item)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.item, got)
		})
	}
}

func TestUpdate(t *testing.T) {
	current := domain.MenuItem{ID: 9, RestaurantID: 17, Name: "Soup", Price: 250, AvailableQuantity: 3}
	want := current
	want.Name = "New soup"
	storageErr := errors.New("update failed")
	applyErr := errors.New("invalid update")
	unexpectedApplyErr := errors.New("apply must not be called")
	type in struct {
		id           domain.MenuItemID
		restaurantID domain.RestaurantID
		apply        func(*domain.MenuItem) error
	}
	type out struct {
		item *domain.MenuItem
		err  error
	}
	updateTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemRepository)
	}{
		{
			name: "updates locked item without losing other fields",
			setup: func(items *mocks.MockMenuItemRepository) {
				copy := current
				gomock.InOrder(
					items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&copy, nil),
					items.EXPECT().Update(gomock.Any(), domain.MenuItemID(9), want).Return(nil),
				)
			},
			in:  in{id: 9, restaurantID: 17, apply: func(item *domain.MenuItem) error { item.Name = "New soup"; return nil }},
			out: out{item: &want},
		},
		{
			name: "rejects item owned by another restaurant",
			setup: func(items *mocks.MockMenuItemRepository) {
				other := current
				other.RestaurantID = 18
				items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&other, nil)
			},
			in:  in{id: 9, restaurantID: 17, apply: func(*domain.MenuItem) error { return unexpectedApplyErr }},
			out: out{err: service.ErrMenuItemNotFound},
		},
		{
			name: "rejects missing item",
			setup: func(items *mocks.MockMenuItemRepository) {
				items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(nil, pgx.ErrNoRows)
			},
			in:  in{id: 9, restaurantID: 17, apply: func(*domain.MenuItem) error { return unexpectedApplyErr }},
			out: out{err: service.ErrMenuItemNotFound},
		},
		{
			name: "does not write when validation fails",
			setup: func(items *mocks.MockMenuItemRepository) {
				copy := current
				items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&copy, nil)
			},
			in:  in{id: 9, restaurantID: 17, apply: func(*domain.MenuItem) error { return applyErr }},
			out: out{err: applyErr},
		},
		{
			name: "propagates update failure",
			setup: func(items *mocks.MockMenuItemRepository) {
				copy := current
				gomock.InOrder(
					items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&copy, nil),
					items.EXPECT().Update(gomock.Any(), domain.MenuItemID(9), want).Return(storageErr),
				)
			},
			in:  in{id: 9, restaurantID: 17, apply: func(item *domain.MenuItem) error { item.Name = "New soup"; return nil }},
			out: out{err: storageErr},
		},
	}

	for _, tt := range updateTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			items := mocks.NewMockMenuItemRepository(ctrl)
			tt.setup(items)
			got, err := menuservice.NewMenuItemService(items, nil, passthroughTxManager{}).
				Update(context.Background(), tt.in.id, tt.in.restaurantID, tt.in.apply)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.item, got)
		})
	}
}

func TestDelete(t *testing.T) {
	storageErr := errors.New("delete failed")
	type in struct {
		id           domain.MenuItemID
		restaurantID domain.RestaurantID
	}
	type out struct{ err error }
	deleteTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemRepository)
	}{
		{
			name: "deletes owned item",
			in:   in{id: 9, restaurantID: 17},
			out:  out{},
			setup: func(items *mocks.MockMenuItemRepository) {
				gomock.InOrder(
					items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&domain.MenuItem{ID: 9, RestaurantID: 17}, nil),
					items.EXPECT().Delete(gomock.Any(), domain.MenuItemID(9)).Return(nil),
				)
			},
		},
		{
			name: "does not delete another restaurant's item",
			in:   in{id: 9, restaurantID: 17},
			setup: func(items *mocks.MockMenuItemRepository) {
				items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&domain.MenuItem{ID: 9, RestaurantID: 18}, nil)
			},
			out: out{err: service.ErrMenuItemNotFound},
		},
		{
			name: "propagates delete failure",
			in:   in{id: 9, restaurantID: 17},
			setup: func(items *mocks.MockMenuItemRepository) {
				gomock.InOrder(
					items.EXPECT().GetByIDForUpdate(gomock.Any(), domain.MenuItemID(9)).Return(&domain.MenuItem{ID: 9, RestaurantID: 17}, nil),
					items.EXPECT().Delete(gomock.Any(), domain.MenuItemID(9)).Return(storageErr),
				)
			},
			out: out{err: storageErr},
		},
	}

	for _, tt := range deleteTest {
		t.Run(tt.name, func(t *testing.T) {
			items := mocks.NewMockMenuItemRepository(gomock.NewController(t))
			tt.setup(items)
			err := menuservice.NewMenuItemService(items, nil, passthroughTxManager{}).Delete(context.Background(), tt.in.id, tt.in.restaurantID)
			require.ErrorIs(t, err, tt.out.err)
		})
	}
}
