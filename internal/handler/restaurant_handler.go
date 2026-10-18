package handler

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/partnerauth"
	"MyKitchen/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RestaurantHandler struct {
	restaurantService RestaurantService
	menuItemService   MenuItemService
}

func NewRestaurantHandler(
	restaurantService RestaurantService,
	menuItemService MenuItemService,
) *RestaurantHandler {
	return &RestaurantHandler{
		restaurantService: restaurantService,
		menuItemService:   menuItemService,
	}
}

func (h *RestaurantHandler) GetAllRestaurants(c *gin.Context) {
	restaurants, err := h.restaurantService.GetRestaurants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})
		return
	}

	response := make([]dto.RestaurantResponse, 0, len(restaurants))
	for _, restaurant := range restaurants {
		response = append(response, dto.NewRestaurantResponse(restaurant))
	}

	c.JSON(http.StatusOK, response)
}

func (h *RestaurantHandler) GetRestaurantByID(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	restaurant, err := h.restaurantService.GetRestaurantByID(c.Request.Context(), restaurantID)
	if err != nil {
		if errors.Is(err, service.ErrRestaurantNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан не найден"})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, dto.NewRestaurantResponse(*restaurant))
}

func (h *RestaurantHandler) RegisterPartnerRestaurant(c *gin.Context) {
	if !acceptJSON(c) {
		return
	}
	var request dto.RegisterRestaurantRequest
	if err := c.ShouldBindJSON(&request); err != nil || !isValidRegisterRestaurantRequest(request) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные ресторана"})
		return
	}

	restaurant, err := h.restaurantService.RegisterRestaurant(c.Request.Context(), domain.Restaurant{
		PartnerID:   partnerauth.PartnerID(c.Request.Context()),
		Name:        strings.TrimSpace(request.Name),
		Phone:       strings.TrimSpace(request.Phone),
		Address:     strings.TrimSpace(request.Address),
		OpeningTime: request.OpeningTime,
		ClosingTime: request.ClosingTime,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось зарегистрировать ресторан"})
		return
	}

	c.JSON(http.StatusCreated, dto.NewRestaurantResponse(*restaurant))
}

func (h *RestaurantHandler) DeletePartnerRestaurant(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	deleted, err := h.restaurantService.DeleteRestaurant(c.Request.Context(), restaurantID)
	if err != nil {
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "не удалось удалить ресторан"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан не найден"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RestaurantHandler) UpdatePartnerAcceptingOrders(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	if !acceptJSON(c) {
		return
	}
	var request dto.UpdateAcceptingOrdersRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.IsAcceptingOrders == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные для изменения приёма заказов"})
		return
	}

	updated, err := h.restaurantService.UpdateAcceptingOrders(
		c.Request.Context(),
		restaurantID,
		*request.IsAcceptingOrders,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось изменить приём заказов"})
		return
	}
	if !updated {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан не найден"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RestaurantHandler) CreatePartnerMenuItem(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	if !acceptJSON(c) {
		return
	}
	var request dto.CreateMenuItemRequest
	if err := c.ShouldBindJSON(&request); err != nil || !isValidCreateMenuItemRequest(request) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные позиции меню"})
		return
	}

	menuItem, err := h.menuItemService.Add(c.Request.Context(), domain.MenuItem{
		Name:              strings.TrimSpace(request.Name),
		Description:       strings.TrimSpace(request.Description),
		Price:             request.Price,
		IsAvailable:       request.IsAvailable,
		AvailableQuantity: request.AvailableQuantity,
		RestaurantID:      restaurantID,
	})
	if err != nil {
		if errors.Is(err, service.ErrRestaurantNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан не найден"})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось создать позицию меню"})
		return
	}

	c.JSON(http.StatusCreated, newPartnerMenuItemResponse(*menuItem))
}

func (h *RestaurantHandler) UpdatePartnerMenuItem(c *gin.Context) {
	restaurantID, menuItemID, ok := partnerMenuItemIDs(c)
	if !ok {
		return
	}

	if !acceptJSON(c) {
		return
	}
	var request dto.UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные позиции меню"})
		return
	}
	menuItem, err := h.menuItemService.Update(c.Request.Context(), menuItemID, restaurantID, func(item *domain.MenuItem) error {
		if !applyMenuItemUpdate(item, request) {
			return errInvalidMenuItemUpdate
		}
		return nil
	})
	if errors.Is(err, errInvalidMenuItemUpdate) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные позиции меню"})
		return
	}
	if errors.Is(err, service.ErrMenuItemNotFound) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "позиция меню не найдена"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось обновить позицию меню"})
		return
	}

	c.JSON(http.StatusOK, newPartnerMenuItemResponse(*menuItem))
}

func (h *RestaurantHandler) DeletePartnerMenuItem(c *gin.Context) {
	restaurantID, menuItemID, ok := partnerMenuItemIDs(c)
	if !ok {
		return
	}
	if err := h.menuItemService.Delete(c.Request.Context(), menuItemID, restaurantID); errors.Is(err, service.ErrMenuItemNotFound) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "позиция меню не найдена"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось удалить позицию меню"})
		return
	}

	c.Status(http.StatusNoContent)
}

func partnerMenuItemIDs(c *gin.Context) (domain.RestaurantID, domain.MenuItemID, bool) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return 0, 0, false
	}
	menuItemID, err := parseMenuItemID(c.Param("menu_item_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор позиции меню"})
		return 0, 0, false
	}
	return restaurantID, menuItemID, true
}

var errInvalidMenuItemUpdate = errors.New("invalid menu item update")

func isValidRegisterRestaurantRequest(request dto.RegisterRestaurantRequest) bool {
	return strings.TrimSpace(request.Name) != "" &&
		strings.TrimSpace(request.Phone) != "" &&
		strings.TrimSpace(request.Address) != "" &&
		request.OpeningTime != "" &&
		request.ClosingTime != ""
}

func isValidCreateMenuItemRequest(request dto.CreateMenuItemRequest) bool {
	return strings.TrimSpace(request.Name) != "" &&
		strings.TrimSpace(request.Description) != "" &&
		request.Price >= 0 &&
		request.AvailableQuantity >= 0
}

func applyMenuItemUpdate(menuItem *domain.MenuItem, request dto.UpdateMenuItemRequest) bool {
	if request.Name == nil && request.Description == nil && request.Price == nil &&
		request.IsAvailable == nil && request.AvailableQuantity == nil {
		return false
	}
	if request.Name != nil {
		menuItem.Name = strings.TrimSpace(*request.Name)
		if menuItem.Name == "" {
			return false
		}
	}
	if request.Description != nil {
		menuItem.Description = strings.TrimSpace(*request.Description)
		if menuItem.Description == "" {
			return false
		}
	}
	if request.Price != nil {
		if *request.Price < 0 {
			return false
		}
		menuItem.Price = *request.Price
	}
	if request.IsAvailable != nil {
		menuItem.IsAvailable = *request.IsAvailable
	}
	if request.AvailableQuantity != nil {
		if *request.AvailableQuantity < 0 {
			return false
		}
		menuItem.AvailableQuantity = *request.AvailableQuantity
	}

	return true
}

func newPartnerMenuItemResponse(menuItem domain.MenuItem) dto.PartnerMenuItemResponse {
	return dto.PartnerMenuItemResponse{
		ID:                int64(menuItem.ID),
		Name:              menuItem.Name,
		Description:       menuItem.Description,
		Price:             menuItem.Price,
		IsAvailable:       menuItem.IsAvailable,
		AvailableQuantity: menuItem.AvailableQuantity,
	}
}

func parseRestaurantID(value string) (domain.RestaurantID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("некорректный идентификатор ресторана")
	}

	return domain.RestaurantID(id), nil
}

func parseMenuItemID(value string) (domain.MenuItemID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("некорректный идентификатор позиции меню")
	}

	return domain.MenuItemID(id), nil
}
