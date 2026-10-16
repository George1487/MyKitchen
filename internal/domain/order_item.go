package domain

type OrderItemID uint64

type OrderItem struct {
	ID OrderItemID

	OrderID OrderID

	MenuItemID MenuItemID

	Price int64

	Quantity int64
}
