package handler

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService OrderService
}

func NewOrderHandler(orderService OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	if !acceptJSON(c) {
		return
	}
	var request dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil || !isValidCreateOrderRequest(request) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные заказа"})
		return
	}

	order := domain.Order{
		ClientID:     domain.ClientID(request.ClientID),
		RestaurantID: domain.RestaurantID(request.RestaurantID),
		Address:      request.Address,
		Status:       domain.OrderStatusNew,
		OrderItems:   make([]domain.OrderItem, 0, len(request.Items)),
	}
	for _, item := range request.Items {
		order.OrderItems = append(order.OrderItems, domain.OrderItem{
			MenuItemID: domain.MenuItemID(item.MenuItemID),
			Quantity:   item.Quantity,
		})
	}

	createdOrder, err := h.orderService.AddOrder(c.Request.Context(), order)
	if err != nil {
		if errors.Is(err, service.ErrRestaurantNotFound) || errors.Is(err, service.ErrMenuItemNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "ресторан или позиция меню не найдены"})
			return
		}

		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "не удалось создать заказ"})
		return
	}

	c.JSON(http.StatusCreated, dto.NewOrderResponse(*createdOrder))
}

func (h *OrderHandler) GetRestaurantOrders(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}

	orders, err := h.orderService.GetOrdersByRestaurantID(c.Request.Context(), restaurantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})
		return
	}

	response := make([]dto.OrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, dto.NewOrderResponse(order))
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) ChangeRestaurantOrderStatus(c *gin.Context) {
	restaurantID, err := parseRestaurantID(c.Param("restaurant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})
		return
	}
	orderID, err := parseOrderID(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный идентификатор заказа"})
		return
	}

	if !acceptJSON(c) {
		return
	}
	var request dto.ChangeOrderStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректные данные статуса заказа"})
		return
	}

	err = h.orderService.ChangeStatusForRestaurant(
		c.Request.Context(),
		restaurantID,
		orderID,
		domain.OrderStatus(request.Status),
	)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "заказ не найден"})
			return
		}

		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "не удалось изменить статус заказа"})
		return
	}

	c.Status(http.StatusNoContent)
}

func isValidCreateOrderRequest(request dto.CreateOrderRequest) bool {
	if request.ClientID <= 0 || request.RestaurantID <= 0 || strings.TrimSpace(request.Address) == "" || len(request.Items) == 0 {
		return false
	}

	for _, item := range request.Items {
		if item.MenuItemID <= 0 || item.Quantity <= 0 {
			return false
		}
	}

	return true
}

func parseOrderID(value string) (domain.OrderID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("некорректный идентификатор заказа")
	}

	return domain.OrderID(id), nil
}
