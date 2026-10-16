package domain

import "time"

type OrderID int64
type OrderStatus string

type ClientID int64

const (
	OrderStatusNew OrderStatus = "new"

	OrderStatusAccepted OrderStatus = "accepted"

	OrderStatusCancelled OrderStatus = "cancelled"

	OrderStatusDelivered OrderStatus = "delivered"
)

type Order struct {
	ID OrderID

	ClientID ClientID

	RestaurantID RestaurantID

	Address string

	TotalPrice int64

	Status OrderStatus

	CreatedAt time.Time

	OrderItems []OrderItem
}
