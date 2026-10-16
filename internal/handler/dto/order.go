package dto

import (
	"MyKitchen/internal/domain"
	"time"
)

type CreateOrderRequest struct {
	ClientID     int64                    `json:"client_id"`
	RestaurantID int64                    `json:"restaurant_id"`
	Address      string                   `json:"address"`
	Items        []CreateOrderItemRequest `json:"items"`
}

type CreateOrderItemRequest struct {
	MenuItemID int64 `json:"menu_item_id"`
	Quantity   int64 `json:"quantity"`
}

type ChangeOrderStatusRequest struct {
	Status string `json:"status"`
}

type OrderResponse struct {
	ID           int64               `json:"id"`
	RestaurantID int64               `json:"restaurant_id"`
	Address      string              `json:"address"`
	TotalPrice   int64               `json:"total_price"`
	Status       domain.OrderStatus  `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
	Items        []OrderItemResponse `json:"items"`
}

type OrderItemResponse struct {
	MenuItemID int64 `json:"menu_item_id"`
	Price      int64 `json:"price"`
	Quantity   int64 `json:"quantity"`
}

func NewOrderResponse(order domain.Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(order.OrderItems))
	for _, item := range order.OrderItems {
		items = append(items, OrderItemResponse{
			MenuItemID: int64(item.MenuItemID),
			Price:      item.Price,
			Quantity:   item.Quantity,
		})
	}

	return OrderResponse{
		ID:           int64(order.ID),
		RestaurantID: int64(order.RestaurantID),
		Address:      order.Address,
		TotalPrice:   order.TotalPrice,
		Status:       order.Status,
		CreatedAt:    order.CreatedAt,
		Items:        items,
	}
}
