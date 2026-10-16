//go:build integration

package integration_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository/sql"
	"MyKitchen/internal/service"
	"MyKitchen/migrations"
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrderFlowCreatesOrderAndReservesStock(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	restaurantID, pizzaID, drinkID := createTestRestaurant(t, ctx, pool)

	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	orderRepo := sql.NewOrderSqlRepository(pool)
	orderItemRepo := sql.NewOrderItemSqlRepository(pool)
	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	orderService := service.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo)

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
	if err != nil {
		t.Fatalf("AddOrder() error = %v", err)
	}

	assertAvailableQuantity(t, ctx, menuItemRepo, pizzaID, 4)
	assertAvailableQuantity(t, ctx, menuItemRepo, drinkID, 3)

	orders, err := orderRepo.GetOrdersByRestaurantID(ctx, restaurantID)
	if err != nil {
		t.Fatalf("GetOrdersByRestaurantID() error = %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders))
	}
	if orders[0].Status != domain.OrderStatusNew || orders[0].TotalPrice != 550 {
		t.Errorf("order = %+v, want new order with total price 550", orders[0])
	}
}

func TestOrderFlowCancellationRestoresStock(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t, ctx)
	restaurantID, pizzaID, _ := createTestRestaurant(t, ctx, pool)

	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	orderRepo := sql.NewOrderSqlRepository(pool)
	orderItemRepo := sql.NewOrderItemSqlRepository(pool)
	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	orderService := service.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo)

	_, err := orderService.AddOrder(ctx, domain.Order{
		ClientID:     42,
		RestaurantID: restaurantID,
		Address:      "Москва, ул. Тверская, 1",
		Status:       domain.OrderStatusNew,
		OrderItems:   []domain.OrderItem{{MenuItemID: pizzaID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("AddOrder() error = %v", err)
	}

	orders, err := orderRepo.GetOrdersByRestaurantID(ctx, restaurantID)
	if err != nil || len(orders) != 1 {
		t.Fatalf("GetOrdersByRestaurantID() = %d orders, %v; want one order", len(orders), err)
	}
	if err := orderService.AcceptOrder(ctx, orders[0].ID); err != nil {
		t.Fatalf("AcceptOrder() error = %v", err)
	}
	if err := orderService.CancelOrder(ctx, orders[0].ID); err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}
	cancelledOrder, err := orderRepo.GetByID(ctx, orders[0].ID)
	if err != nil {
		t.Fatalf("GetByID() order error = %v", err)
	}
	if cancelledOrder.Status != domain.OrderStatusCancelled {
		t.Errorf("order status = %q, want %q", cancelledOrder.Status, domain.OrderStatusCancelled)
	}

	assertAvailableQuantity(t, ctx, menuItemRepo, pizzaID, 5)
}

func openTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	if !strings.Contains(strings.ToLower(strings.Trim(parsedURL.Path, "/")), "test") {
		t.Fatal("TEST_DATABASE_URL must point to a database with 'test' in its name")
	}
	if err := migrations.Up(ctx, databaseURL); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	cleanTestData(t, ctx, pool)

	return pool
}

func cleanTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, "TRUNCATE TABLE order_items, orders, menu_items, restaurants RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("clean test data: %v", err)
	}
}

func createTestRestaurant(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (
	domain.RestaurantID,
	domain.MenuItemID,
	domain.MenuItemID,
) {
	t.Helper()

	restaurantRepo := sql.NewRestaurantSqlRepository(pool)
	menuItemRepo := sql.NewMenuItemSqlRepository(pool)
	_, err := restaurantRepo.AddRestaurant(ctx, domain.Restaurant{
		Name:        "Тестовая кухня",
		Phone:       "+79990000000",
		Address:     "Москва",
		OpeningTime: "09:00",
		ClosingTime: "23:00",
	})
	if err != nil {
		t.Fatalf("AddRestaurant() error = %v", err)
	}

	restaurants, err := restaurantRepo.GetAll(ctx)
	if err != nil || len(restaurants) != 1 {
		t.Fatalf("GetAll() = %d restaurants, %v; want one restaurant", len(restaurants), err)
	}
	restaurantID := restaurants[0].ID

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
		if _, err := menuItemRepo.Add(ctx, menuItem); err != nil {
			t.Fatalf("Add() menu item %q error = %v", menuItem.Name, err)
		}
	}
	menuItems, err := menuItemRepo.GetByRestaurantID(ctx, restaurants[0].ID)
	if err != nil || len(menuItems) != 2 {
		t.Fatalf("GetByRestaurantID() = %d items, %v; want two items", len(menuItems), err)
	}

	var pizzaID, drinkID domain.MenuItemID
	for _, menuItem := range menuItems {
		switch menuItem.Name {
		case "Пицца":
			pizzaID = menuItem.ID
		case "Напиток":
			drinkID = menuItem.ID
		}
	}
	if pizzaID == 0 || drinkID == 0 {
		t.Fatalf("created menu items not found: pizzaID=%d, drinkID=%d", pizzaID, drinkID)
	}

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
	if err != nil {
		t.Fatalf("GetByID() menu item error = %v", err)
	}
	if menuItem.AvailableQuantity != want {
		t.Errorf("available quantity = %d, want %d", menuItem.AvailableQuantity, want)
	}
}
