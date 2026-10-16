package dto

import "MyKitchen/internal/domain"

type MenuItemResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	IsAvailable bool   `json:"is_available"`
}

func NewMenuItemResponse(menuItem domain.MenuItem) MenuItemResponse {
	return MenuItemResponse{
		ID:          int64(menuItem.ID),
		Name:        menuItem.Name,
		Description: menuItem.Description,
		Price:       menuItem.Price,
		IsAvailable: menuItem.IsAvailable,
	}
}
