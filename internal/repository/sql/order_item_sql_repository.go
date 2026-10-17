package sql

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"MyKitchen/internal/transaction"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItemSqlRepository struct {
	db *pgxpool.Pool
}

var _ repository.OrderItemRepository = (*OrderItemSqlRepository)(nil)

func NewOrderItemSqlRepository(db *pgxpool.Pool) *OrderItemSqlRepository {
	return &OrderItemSqlRepository{db: db}
}

func (r *OrderItemSqlRepository) GetByID(ctx context.Context, orderItemID domain.OrderItemID) (*domain.OrderItem, error) {

	var orderItem domain.OrderItem

	err := transaction.DB(ctx, r.db).QueryRow(ctx,
		"SELECT id, order_id, menu_item_id, price, quantity FROM order_items WHERE id = $1",
		orderItemID).Scan(
		&orderItem.ID,
		&orderItem.OrderID,
		&orderItem.MenuItemID,
		&orderItem.Price,
		&orderItem.Quantity)

	if err != nil {
		return nil, err
	}

	return &orderItem, nil
}

func (r *OrderItemSqlRepository) GetByOrderID(ctx context.Context, orderID domain.OrderID) ([]domain.OrderItem, error) {

	rows, err := transaction.DB(ctx, r.db).Query(ctx,
		"SELECT id, order_id, menu_item_id, price, quantity FROM order_items WHERE order_id = $1",
		orderID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orderItems []domain.OrderItem

	for rows.Next() {
		var orderItem domain.OrderItem

		err := rows.Scan(
			&orderItem.ID,
			&orderItem.OrderID,
			&orderItem.MenuItemID,
			&orderItem.Price,
			&orderItem.Quantity)

		if err != nil {
			return nil, err
		}

		orderItems = append(orderItems, orderItem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orderItems, nil
}

func (r *OrderItemSqlRepository) AddOrderItem(ctx context.Context, orderItem domain.OrderItem) error {

	_, err := transaction.DB(ctx, r.db).Exec(ctx,
		"INSERT INTO order_items (order_id, menu_item_id, price, quantity) VALUES ($1, $2, $3, $4)",
		orderItem.OrderID,
		orderItem.MenuItemID,
		orderItem.Price,
		orderItem.Quantity)

	return err
}

func (r *OrderItemSqlRepository) DeleteOrderItem(ctx context.Context, orderItemID domain.OrderItemID) error {

	_, err := transaction.DB(ctx, r.db).Exec(ctx,
		"DELETE FROM order_items WHERE id = $1",
		orderItemID)

	return err
}
