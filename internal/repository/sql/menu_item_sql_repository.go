package sql

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
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

	var item domain.MenuItem

	err := r.db.QueryRow(ctx, "SELECT id, name, description, price, is_available, available_quantity, restaurant_id FROM menu_items WHERE id = $1", id).Scan(
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

	rows, err := r.db.
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

	err := r.db.QueryRow(ctx,
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

	_, err := r.db.Exec(ctx, "DELETE FROM menu_items WHERE id = $1", id)

	return err
}

func (r *MenuItemSqlRepository) Update(ctx context.Context, id domain.MenuItemID, item domain.MenuItem) error {

	_, err := r.db.Exec(ctx,
		"UPDATE menu_items SET name = $1, description = $2, price = $3, is_available = $4, available_quantity = $5, restaurant_id = $6 WHERE id = $7",
		item.Name,
		item.Description,
		item.Price,
		item.IsAvailable,
		item.AvailableQuantity,
		item.RestaurantID,
		id)

	return err
}
