package domain

type RestaurantID int64

type Restaurant struct {
	ID RestaurantID

	Name string

	Address string

	Phone string

	OpeningTime string

	ClosingTime string

	IsAcceptingOrders bool

	MenuItems []MenuItem
}
