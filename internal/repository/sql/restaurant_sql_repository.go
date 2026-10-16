package sql

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/repository"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RestaurantSqlRepository struct {
	db *pgxpool.Pool
}

var _ repository.RestaurantRepository = (*RestaurantSqlRepository)(nil)

func NewRestaurantSqlRepository(db *pgxpool.Pool) *RestaurantSqlRepository {
	return &RestaurantSqlRepository{db: db}
}

func (r *RestaurantSqlRepository) GetByID(ctx context.Context, restaurantID domain.RestaurantID) (*domain.Restaurant, error) {

	var restaurant domain.Restaurant

	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, address, opening_time, closing_time, is_accepting_orders
		 FROM restaurants
		 WHERE id = $1`, restaurantID).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.Phone,
		&restaurant.Address,
		&restaurant.OpeningTime,
		&restaurant.ClosingTime,
		&restaurant.IsAcceptingOrders)

	if err != nil {
		return nil, err
	}

	return &restaurant, nil
}

func (r *RestaurantSqlRepository) AddRestaurant(ctx context.Context, restaurant domain.Restaurant) (domain.RestaurantID, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO restaurants
            (name, phone, address, opening_time, closing_time)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id`,
		restaurant.Name,
		restaurant.Phone,
		restaurant.Address,
		restaurant.OpeningTime,
		restaurant.ClosingTime,
	).Scan(&restaurant.ID)

	if err != nil {
		return 0, err
	}

	for _, item := range restaurant.MenuItems {
		_, err = tx.Exec(ctx,
			`INSERT INTO menu_items
                (name, description, price, is_available, available_quantity, restaurant_id)
             VALUES ($1, $2, $3, $4, $5, $6)`,
			item.Name,
			item.Description,
			item.Price,
			item.IsAvailable,
			item.AvailableQuantity,
			restaurant.ID,
		)

		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return restaurant.ID, nil
}

func (r *RestaurantSqlRepository) DeleteRestaurant(ctx context.Context, restaurantID domain.RestaurantID) (bool, error) {
	result, err := r.db.Exec(ctx, "DELETE FROM restaurants WHERE id = $1", restaurantID)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

func (r *RestaurantSqlRepository) UpdateAcceptingOrders(
	ctx context.Context,
	restaurantID domain.RestaurantID,
	isAcceptingOrders bool,
) (bool, error) {
	result, err := r.db.Exec(ctx,
		"UPDATE restaurants SET is_accepting_orders = $1 WHERE id = $2",
		isAcceptingOrders,
		restaurantID,
	)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

func (r *RestaurantSqlRepository) GetAll(ctx context.Context) ([]domain.Restaurant, error) {

	var restaurants []domain.Restaurant

	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, address, opening_time, closing_time, is_accepting_orders
		 FROM restaurants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant domain.Restaurant

		err := rows.Scan(&restaurant.ID,
			&restaurant.Name,
			&restaurant.Phone,
			&restaurant.Address,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.IsAcceptingOrders)

		if err != nil {
			return nil, err
		}

		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}
