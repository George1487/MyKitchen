package handler

import (
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MenuItemHandler struct {
	menuItemService *service.MenuItemService
}

func NewMenuItemHandler(menuItemService *service.MenuItemService) *MenuItemHandler {
	return &MenuItemHandler{menuItemService: menuItemService}
}

func (h *MenuItemHandler) GetMenuItemsByRestaurantID(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	menuItems, err := h.menuItemService.GetMenuItemsByRestaurantID(c.Request.Context(), restaurantID)
	if err != nil {
		if errors.Is(err, service.ErrRestaurantNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан не найден"})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})
		return
	}

	response := make([]dto.MenuItemResponse, 0, len(menuItems))
	for _, menuItem := range menuItems {
		response = append(response, dto.NewMenuItemResponse(menuItem))
	}

	c.JSON(http.StatusOK, response)
}
