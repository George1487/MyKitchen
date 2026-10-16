package dto

import "MyKitchen/internal/domain"

type RestaurantResponse struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Address           string `json:"address"`
	Phone             string `json:"phone"`
	OpeningTime       string `json:"opening_time"`
	ClosingTime       string `json:"closing_time"`
	IsAcceptingOrders bool   `json:"is_accepting_orders"`
}

func NewRestaurantResponse(restaurant domain.Restaurant) RestaurantResponse {
	return RestaurantResponse{
		ID:                int64(restaurant.ID),
		Name:              restaurant.Name,
		Address:           restaurant.Address,
		Phone:             restaurant.Phone,
		OpeningTime:       restaurant.OpeningTime,
		ClosingTime:       restaurant.ClosingTime,
		IsAcceptingOrders: restaurant.IsAcceptingOrders,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}
