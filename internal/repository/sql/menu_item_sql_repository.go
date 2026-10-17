package sql

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"MyKitchen/internal/transaction"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuItemSqlRepository struct {
	db *pgxpool.Pool
}

func NewMenuItemSqlRepository(db *pgxpool.Pool) *MenuItemSqlRepository {

	return &MenuItemSqlRepository{db: db}
}

var _ repository.MenuItemRepository = (*MenuItemSqlRepository)(nil)

func (r *MenuItemSqlRepository) GetByID(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error) {
	return r.getByID(ctx, id, false)
}

func (r *MenuItemSqlRepository) GetByIDForUpdate(ctx context.Context, id domain.MenuItemID) (*domain.MenuItem, error) {
	if _, err := transaction.Tx(ctx); err != nil {
		return nil, err
	}
	return r.getByID(ctx, id, true)
}

func (r *MenuItemSqlRepository) getByID(ctx context.Context, id domain.MenuItemID, forUpdate bool) (*domain.MenuItem, error) {

	var item domain.MenuItem
	query := "SELECT id, name, description, price, is_available, available_quantity, restaurant_id FROM menu_items WHERE id = $1"
	if forUpdate {
		query += " FOR UPDATE"
	}
	err := transaction.DB(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.IsAvailable,
		&item.AvailableQuantity,
		&item.RestaurantID)

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *MenuItemSqlRepository) GetByRestaurantID(ctx context.Context, restaurantID domain.RestaurantID) ([]domain.MenuItem, error) {

	rows, err := transaction.DB(ctx, r.db).
		Query(ctx, "SELECT id, name, description, price, is_available, available_quantity,"+
			" restaurant_id FROM menu_items WHERE restaurant_id = $1", restaurantID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menuItems []domain.MenuItem
	for rows.Next() {
		var menuItem domain.MenuItem
		err := rows.Scan(&menuItem.ID, &menuItem.Name, &menuItem.Description, &menuItem.Price,
			&menuItem.IsAvailable, &menuItem.AvailableQuantity, &menuItem.RestaurantID)
		if err != nil {
			return nil, err
		}
		menuItems = append(menuItems, menuItem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return menuItems, nil
}

func (r *MenuItemSqlRepository) Add(ctx context.Context, item domain.MenuItem) (domain.MenuItemID, error) {
	var menuItemID domain.MenuItemID

	err := transaction.DB(ctx, r.db).QueryRow(ctx,
		`INSERT INTO menu_items (name, description, price, is_available, available_quantity, restaurant_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		item.Name,
		item.Description,
		item.Price,
		item.IsAvailable,
		item.AvailableQuantity,
		item.RestaurantID,
	).Scan(&menuItemID)
	if err != nil {
		return 0, err
	}

	return menuItemID, nil
}

func (r *MenuItemSqlRepository) Delete(ctx context.Context, id domain.MenuItemID) error {

	_, err := transaction.DB(ctx, r.db).Exec(ctx, "DELETE FROM menu_items WHERE id = $1", id)

	return err
}

func (r *MenuItemSqlRepository) Update(ctx context.Context, id domain.MenuItemID, item domain.MenuItem) error {

	_, err := transaction.DB(ctx, r.db).Exec(ctx,
		"UPDATE menu_items SET name = $1, description = $2, price = $3, is_available = $4, available_quantity = $5 WHERE id = $6",
		item.Name,
		item.Description,
		item.Price,
		item.IsAvailable,
		item.AvailableQuantity,
		id)

	return err
}
