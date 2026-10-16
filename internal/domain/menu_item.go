package domain

type MenuItemID int64

type MenuItem struct {
	ID MenuItemID

	Name string

	Description string

	Price int64

	IsAvailable bool

	AvailableQuantity int64

	RestaurantID RestaurantID
}
