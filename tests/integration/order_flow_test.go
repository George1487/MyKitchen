//go:build integrational

package integration_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository/sql"
	orderservice "MyKitchen/internal/service/order"
	"MyKitchen/internal/transaction"
	"MyKitchen/migrations"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderFlowCreatesOrderAndReservesStock(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	restaurantID, pizzaID, drinkID := createTestRestaurant(t, ctx, pool)

	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	orderRepo := sql.NewOrderSqlRepository(pool)
	orderItemRepo := sql.NewOrderItemSqlRepository(pool)
	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	orderService := orderservice.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo, transaction.NewPgxTxManager(pool))

	_, err := orderService.AddOrder(ctx, domain.Order{
		ClientID:     42,
		RestaurantID: restaurantID,
		Address:      "Москва, ул. Тверская, 1",
		Status:       domain.OrderStatusNew,
		OrderItems: []domain.OrderItem{
			{MenuItemID: pizzaID, Quantity: 1},
			{MenuItemID: drinkID, Quantity: 2},
		},
	})
	require.NoError(t, err)

	assertAvailableQuantity(t, ctx, menuItemRepo, pizzaID, 4)
	assertAvailableQuantity(t, ctx, menuItemRepo, drinkID, 3)

	orders, err := orderRepo.GetOrdersByRestaurantID(ctx, restaurantID)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, domain.OrderStatusNew, orders[0].Status)
	require.Equal(t, int64(550), orders[0].TotalPrice)
}

func TestOrderFlowCancellationRestoresStock(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	restaurantID, pizzaID, _ := createTestRestaurant(t, ctx, pool)

	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	orderRepo := sql.NewOrderSqlRepository(pool)
	orderItemRepo := sql.NewOrderItemSqlRepository(pool)
	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	orderService := orderservice.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo, transaction.NewPgxTxManager(pool))

	_, err := orderService.AddOrder(ctx, domain.Order{
		ClientID:     42,
		RestaurantID: restaurantID,
		Address:      "Москва, ул. Тверская, 1",
		Status:       domain.OrderStatusNew,
		OrderItems:   []domain.OrderItem{{MenuItemID: pizzaID, Quantity: 2}},
	})
	require.NoError(t, err)

	orders, err := orderRepo.GetOrdersByRestaurantID(ctx, restaurantID)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.NoError(t, orderService.AcceptOrder(ctx, orders[0].ID))
	require.NoError(t, orderService.CancelOrder(ctx, orders[0].ID))
	cancelledOrder, err := orderRepo.GetByID(ctx, orders[0].ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, cancelledOrder.Status)

	assertAvailableQuantity(t, ctx, menuItemRepo, pizzaID, 5)
}

func TestTransactionManagerReadCommittedAndRollback(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	manager := transaction.NewPgxTxManager(pool)
	restaurants := sql.NewRestaurantSqlRepository(pool)
	wantErr := errors.New("rollback requested")

	err := manager.Do(ctx, func(txCtx context.Context) error {
		var isolation string
		assert.NoError(t, transaction.DB(txCtx, pool).QueryRow(txCtx, "SHOW transaction_isolation").Scan(&isolation))
		assert.Equal(t, "read committed", isolation)
		_, insertErr := restaurants.AddRestaurant(txCtx, domain.Restaurant{
			Name: "Rollback kitchen", Phone: "+79990000000", Address: "Moscow",
			OpeningTime: "09:00", ClosingTime: "23:00",
		})
		assert.NoError(t, insertErr)
		return wantErr
	})
	require.ErrorIs(t, err, wantErr)
	rows, err := restaurants.GetAll(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestConcurrentOrdersDoNotOversell(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	restaurantID, pizzaID, _ := createTestRestaurant(t, ctx, pool)
	menuItems := sql.NewMenuItemSqlRepository(pool)
	orders := sql.NewOrderSqlRepository(pool)
	orderService := orderservice.NewOrderService(orders, menuItems, sql.NewOrderItemSqlRepository(pool),
		sql.NewRestaurantSqlRepository(pool), transaction.NewPgxTxManager(pool))

	results := make(chan error, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := orderService.AddOrder(ctx, domain.Order{
				ClientID: 42, RestaurantID: restaurantID, Address: "Moscow", Status: domain.OrderStatusNew,
				OrderItems: []domain.OrderItem{{MenuItemID: pizzaID, Quantity: 4}},
			})
			results <- err
		}()
	}
	workers.Wait()
	firstErr, secondErr := <-results, <-results
	require.NotEqual(t, firstErr == nil, secondErr == nil, "exactly one order should succeed")
	assertAvailableQuantity(t, ctx, menuItems, pizzaID, 1)
	created, err := orders.GetOrdersByRestaurantID(ctx, restaurantID)
	require.NoError(t, err)
	require.Len(t, created, 1)
}

func openTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	dockerPool := dockertest.NewPoolT(t, "")
	postgres := dockerPool.RunT(t, "postgres",
		dockertest.WithTag("16-alpine"),
		dockertest.WithEnv([]string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=test_password",
			"POSTGRES_DB=mykitchen_test",
		}),
		dockertest.WithoutReuse(),
	)
	hostPort := postgres.GetHostPort("5432/tcp")
	require.NotEmpty(t, hostPort)
	databaseURL := fmt.Sprintf("postgres://postgres:test_password@%s/mykitchen_test?sslmode=disable", hostPort)

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, dockerPool.Retry(ctx, 90*time.Second, func() error {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return pool.Ping(pingCtx)
	}))
	require.NoError(t, migrations.Up(ctx, databaseURL))

	return pool
}

func createTestRestaurant(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (
	domain.RestaurantID,
	domain.MenuItemID,
	domain.MenuItemID,
) {
	t.Helper()

	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	var restaurantID domain.RestaurantID
	err := transaction.NewPgxTxManager(pool).Do(ctx, func(txCtx context.Context) error {
		var err error
		restaurantID, err = restaurantRepo.AddRestaurant(txCtx, domain.Restaurant{
			Name:        "Тестовая кухня",
			Phone:       "+79990000000",
			Address:     "Москва",
			OpeningTime: "09:00",
			ClosingTime: "23:00",
		})
		return err
	})
	require.NoError(t, err)

	restaurants, err := restaurantRepo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, restaurants, 1)
	restaurantID = restaurants[0].ID

	for _, menuItem := range []domain.MenuItem{
		{
			Name:              "Пицца",
			Description:       "Тестовая пицца",
			Price:             350,
			IsAvailable:       true,
			AvailableQuantity: 5,
			RestaurantID:      restaurantID,
		},
		{
			Name:              "Напиток",
			Description:       "Тестовый напиток",
			Price:             100,
			IsAvailable:       true,
			AvailableQuantity: 5,
			RestaurantID:      restaurantID,
		},
	} {
		_, err := menuItemRepo.Add(ctx, menuItem)
		require.NoError(t, err)
	}
	menuItems, err := menuItemRepo.GetByRestaurantID(ctx, restaurants[0].ID)
	require.NoError(t, err)
	require.Len(t, menuItems, 2)

	var pizzaID, drinkID domain.MenuItemID
	for _, menuItem := range menuItems {
		switch menuItem.Name {
		case "Пицца":
			pizzaID = menuItem.ID
		case "Напиток":
			drinkID = menuItem.ID
		}
	}
	require.NotZero(t, pizzaID)
	require.NotZero(t, drinkID)

	return restaurantID, pizzaID, drinkID
}

func assertAvailableQuantity(
	t *testing.T,
	ctx context.Context,
	menuItemRepo *sql.MenuItemSqlRepository,
	menuItemID domain.MenuItemID,
	want int64,
) {
	t.Helper()

	menuItem, err := menuItemRepo.GetByID(ctx, menuItemID)
	require.NoError(t, err)
	require.Equal(t, want, menuItem.AvailableQuantity)
}
