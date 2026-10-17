package restaurant_test

import (
	"context"
	"errors"
	"testing"

	"MyKitchen/internal/domain"
	"MyKitchen/internal/service"
	restaurantservice "MyKitchen/internal/service/restaurant"
	"MyKitchen/tests/mocks"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type passthroughTxManager struct{}

func (passthroughTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestRegisterRestaurant(t *testing.T) {
	input := domain.Restaurant{Name: "Kitchen", Phone: "+79990000000", Address: "Moscow"}
	created := input
	created.ID = 17
	created.IsAcceptingOrders = true
	storageErr := errors.New("insert failed")
	type in struct{ restaurant domain.Restaurant }
	type out struct {
		restaurant *domain.Restaurant
		err        error
	}
	registerRestaurantTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantRepository)
	}{
		{
			name: "returns created restaurant and accepting status",
			in:   in{restaurant: input},
			setup: func(repo *mocks.MockRestaurantRepository) {
				repo.EXPECT().AddRestaurant(gomock.Any(), input).Return(domain.RestaurantID(17), nil)
			},
			out: out{restaurant: &created},
		},
		{
			name: "does not return restaurant after insert failure",
			in:   in{restaurant: input},
			setup: func(repo *mocks.MockRestaurantRepository) {
				repo.EXPECT().AddRestaurant(gomock.Any(), input).Return(domain.RestaurantID(0), storageErr)
			},
			out: out{err: storageErr},
		},
	}

	for _, tt := range registerRestaurantTest {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockRestaurantRepository(gomock.NewController(t))
			tt.setup(repo)
			got, err := restaurantservice.NewRestaurantService(repo, passthroughTxManager{}).
				RegisterRestaurant(context.Background(), tt.in.restaurant)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.restaurant, got)
		})
	}
}

func TestGetRestaurantByID(t *testing.T) {
	storageErr := errors.New("read failed")
	type in struct {
		id             domain.RestaurantID
		repoRestaurant *domain.Restaurant
		repoErr        error
	}
	type out struct {
		restaurant *domain.Restaurant
		err        error
	}
	getRestaurantByIDTest := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "returns restaurant",
			in:   in{id: 17, repoRestaurant: &domain.Restaurant{ID: 17, Name: "Kitchen"}},
			out:  out{restaurant: &domain.Restaurant{ID: 17, Name: "Kitchen"}},
		},
		{name: "maps missing row to domain error", in: in{id: 17, repoErr: pgx.ErrNoRows}, out: out{err: service.ErrRestaurantNotFound}},
		{name: "preserves storage error", in: in{id: 17, repoErr: storageErr}, out: out{err: storageErr}},
	}

	for _, tt := range getRestaurantByIDTest {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockRestaurantRepository(gomock.NewController(t))
			repo.EXPECT().GetByID(gomock.Any(), tt.in.id).Return(tt.in.repoRestaurant, tt.in.repoErr)
			got, err := restaurantservice.NewRestaurantService(repo, nil).GetRestaurantByID(context.Background(), tt.in.id)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.restaurant, got)
		})
	}
}

func TestUpdateAcceptingOrders(t *testing.T) {
	storageErr := errors.New("update failed")
	type in struct {
		id          domain.RestaurantID
		accepting   bool
		repoUpdated bool
		repoErr     error
	}
	type out struct {
		updated bool
		err     error
	}
	updateAcceptingOrdersTest := []struct {
		name string
		in   in
		out  out
	}{
		{name: "restaurant updated", in: in{id: 17, accepting: false, repoUpdated: true}, out: out{updated: true}},
		{name: "restaurant missing", in: in{id: 17, accepting: false}, out: out{}},
		{name: "storage failure", in: in{id: 17, accepting: false, repoErr: storageErr}, out: out{err: storageErr}},
	}

	for _, tt := range updateAcceptingOrdersTest {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockRestaurantRepository(gomock.NewController(t))
			repo.EXPECT().UpdateAcceptingOrders(gomock.Any(), tt.in.id, tt.in.accepting).Return(tt.in.repoUpdated, tt.in.repoErr)
			got, err := restaurantservice.NewRestaurantService(repo, nil).UpdateAcceptingOrders(context.Background(), tt.in.id, tt.in.accepting)
			require.ErrorIs(t, err, tt.out.err)
			require.Equal(t, tt.out.updated, got)
		})
	}
}
