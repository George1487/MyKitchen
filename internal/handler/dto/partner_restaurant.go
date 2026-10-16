package dto

type RegisterRestaurantRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	OpeningTime string `json:"opening_time"`
	ClosingTime string `json:"closing_time"`
}

type UpdateAcceptingOrdersRequest struct {
	IsAcceptingOrders *bool `json:"is_accepting_orders"`
}

type CreateMenuItemRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Price             int64  `json:"price"`
	IsAvailable       bool   `json:"is_available"`
	AvailableQuantity int64  `json:"available_quantity"`
}

type UpdateMenuItemRequest struct {
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	Price             *int64  `json:"price"`
	IsAvailable       *bool   `json:"is_available"`
	AvailableQuantity *int64  `json:"available_quantity"`
}

type PartnerMenuItemResponse struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Price             int64  `json:"price"`
	IsAvailable       bool   `json:"is_available"`
	AvailableQuantity int64  `json:"available_quantity"`
}
