package sql

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"MyKitchen/internal/transaction"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderSqlRepository struct {
	db *pgxpool.Pool
}

var _ repository.OrderRepository = (*OrderSqlRepository)(nil)

func NewOrderSqlRepository(db *pgxpool.Pool) *OrderSqlRepository {
	return &OrderSqlRepository{db: db}
}

func (r *OrderSqlRepository) GetByID(ctx context.Context, orderID domain.OrderID) (*domain.Order, error) {
	return r.getByID(ctx, orderID, false)
}

func (r *OrderSqlRepository) GetByIDForUpdate(ctx context.Context, orderID domain.OrderID) (*domain.Order, error) {
	if _, err := transaction.Tx(ctx); err != nil {
		return nil, err
	}
	return r.getByID(ctx, orderID, true)
}

func (r *OrderSqlRepository) getByID(ctx context.Context, orderID domain.OrderID, forUpdate bool) (*domain.Order, error) {

	var order domain.Order
	query := `SELECT id, client_id, restaurant_id, total_price, status, address, created_at
		 FROM orders
		 WHERE id = $1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	err := transaction.DB(ctx, r.db).QueryRow(ctx, query, orderID).Scan(
		&order.ID,
		&order.ClientID,
		&order.RestaurantID,
		&order.TotalPrice,
		&order.Status,
		&order.Address,
		&order.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *OrderSqlRepository) GetOrdersByRestaurantID(
	ctx context.Context,
	restaurantID domain.RestaurantID,
) ([]domain.Order, error) {
	rows, err := transaction.DB(ctx, r.db).Query(ctx,
		"SELECT * FROM orders WHERE restaurant_id = $1 ORDER BY created_at DESC", restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.RestaurantID,
			&order.TotalPrice,
			&order.Status,
			&order.Address,
			&order.CreatedAt,
		); err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderSqlRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.OrderID, error) {
	tx, err := transaction.Tx(ctx)
	if err != nil {
		return 0, err
	}

	requestedQuantities := make(map[domain.MenuItemID]int64)
	for _, orderItem := range order.OrderItems {
		if orderItem.Quantity <= 0 {
			return 0, fmt.Errorf("quantity for menu item %d must be positive", orderItem.MenuItemID)
		}

		requestedQuantities[orderItem.MenuItemID] += orderItem.Quantity
	}

	menuItemIDs := make([]domain.MenuItemID, 0, len(requestedQuantities))
	for menuItemID := range requestedQuantities {
		menuItemIDs = append(menuItemIDs, menuItemID)
	}
	sort.Slice(menuItemIDs, func(i, j int) bool {
		return menuItemIDs[i] < menuItemIDs[j]
	})

	for _, menuItemID := range menuItemIDs {
		_, err := tx.Exec(ctx,
			`UPDATE menu_items
			 SET available_quantity = available_quantity - $1
			 WHERE id = $2`,
			requestedQuantities[menuItemID],
			menuItemID,
		)
		if err != nil {
			return 0, fmt.Errorf("reserve menu item %d: %w", menuItemID, err)
		}
	}

	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}

	var orderID domain.OrderID

	err = tx.QueryRow(ctx,
		`INSERT INTO orders
            (client_id, restaurant_id, total_price, status, address, created_at)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id`,
		order.ClientID,
		order.RestaurantID,
		order.TotalPrice,
		order.Status,
		order.Address,
		order.CreatedAt,
	).Scan(&orderID)

	if err != nil {
		return 0, err
	}

	for _, item := range order.OrderItems {
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items
                (order_id, menu_item_id, price, quantity)
             VALUES ($1, $2, $3, $4)`,
			orderID,
			item.MenuItemID,
			item.Price,
			item.Quantity,
		)

		if err != nil {
			return 0, err
		}
	}

	return orderID, nil
}

func (r *OrderSqlRepository) CancelOrder(
	ctx context.Context,
	orderID domain.OrderID,
	fromStatus domain.OrderStatus,
) (bool, error) {
	tx, err := transaction.Tx(ctx)
	if err != nil {
		return false, err
	}

	var updatedOrderID domain.OrderID
	err = tx.QueryRow(ctx,
		`UPDATE orders
		 SET status = $1
		 WHERE id = $2 AND status = $3
		 RETURNING id`,
		domain.OrderStatusCancelled,
		orderID,
		fromStatus,
	).Scan(&updatedOrderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if err := restockOrderItems(ctx, tx, orderID); err != nil {
		return false, err
	}

	return true, nil
}

func restockOrderItems(ctx context.Context, tx pgx.Tx, orderID domain.OrderID) error {
	rows, err := tx.Query(ctx,
		`SELECT menu_item_id, SUM(quantity)
		 FROM order_items
		 WHERE order_id = $1
		 GROUP BY menu_item_id
		 ORDER BY menu_item_id ASC`, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type restockItem struct {
		menuItemID domain.MenuItemID
		quantity   int64
	}
	var restockItems []restockItem
	for rows.Next() {
		var menuItemID domain.MenuItemID
		var quantity int64
		if err := rows.Scan(&menuItemID, &quantity); err != nil {
			return err
		}
		restockItems = append(restockItems, restockItem{menuItemID: menuItemID, quantity: quantity})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	sort.Slice(restockItems, func(i, j int) bool {
		return restockItems[i].menuItemID < restockItems[j].menuItemID
	})

	for _, item := range restockItems {
		_, err := tx.Exec(ctx,
			`UPDATE menu_items
			 SET available_quantity = available_quantity + $1
			 WHERE id = $2`, item.quantity, item.menuItemID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *OrderSqlRepository) UpdateStatus(
	ctx context.Context,
	orderID domain.OrderID,
	fromStatus domain.OrderStatus,
	toStatus domain.OrderStatus) (bool, error) {
	result, err := transaction.DB(ctx, r.db).Exec(
		ctx, "UPDATE orders SET status = $1 WHERE id = $2 AND status = $3",
		toStatus,
		orderID,
		fromStatus,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

func (r *OrderSqlRepository) DeleteOrder(ctx context.Context, orderID domain.OrderID) error {
	tx, err := transaction.Tx(ctx)
	if err != nil {
		return err
	}

	var status domain.OrderStatus
	err = tx.QueryRow(ctx,
		`SELECT status FROM orders WHERE id = $1 FOR UPDATE`,
		orderID,
	).Scan(&status)
	if err != nil {
		return err
	}

	if status == domain.OrderStatusNew || status == domain.OrderStatusAccepted {
		if err := restockOrderItems(ctx, tx, orderID); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
	return err
}
