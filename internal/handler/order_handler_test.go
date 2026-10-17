package handler_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/handler"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/service"
	"MyKitchen/tests/mocks"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestCreateOrder(t *testing.T) {
	storageErr := errors.New("storage failure")
	createdAt := time.Date(2026, time.October, 17, 12, 0, 0, 0, time.UTC)
	created := &domain.Order{
		ID: 11, ClientID: 42, RestaurantID: 17, Address: "Moscow", TotalPrice: 700,
		Status: domain.OrderStatusNew, CreatedAt: createdAt,
		OrderItems: []domain.OrderItem{{MenuItemID: 5, Price: 350, Quantity: 2}},
	}
	validBody := `{"client_id":42,"restaurant_id":17,"address":"Moscow","items":[{"menu_item_id":5,"quantity":2}]}`
	expectedOrder := domain.Order{
		ClientID: 42, RestaurantID: 17, Address: "Moscow", Status: domain.OrderStatusNew,
		OrderItems: []domain.OrderItem{{MenuItemID: 5, Quantity: 2}},
	}
	type in struct{ body string }
	type out struct {
		status int
		body   string
	}
	createOrderTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockOrderService)
	}{
		{
			name: "creates order", in: in{body: validBody},
			out: out{status: http.StatusCreated, body: jsonBody(t, dto.NewOrderResponse(*created))},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().AddOrder(gomock.Any(), expectedOrder).Return(created, nil)
			},
		},
		{
			name: "rejects malformed JSON", in: in{body: "{"},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные заказа"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "rejects zero quantity", in: in{body: `{"client_id":42,"restaurant_id":17,"address":"Moscow","items":[{"menu_item_id":5,"quantity":0}]}`},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные заказа"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "restaurant missing", in: in{body: validBody},
			out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан или позиция меню не найдены"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().AddOrder(gomock.Any(), expectedOrder).Return(nil, service.ErrRestaurantNotFound)
			},
		},
		{
			name: "menu item missing", in: in{body: validBody},
			out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан или позиция меню не найдены"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().AddOrder(gomock.Any(), expectedOrder).Return(nil, service.ErrMenuItemNotFound)
			},
		},
		{
			name: "service failure", in: in{body: validBody},
			out: out{status: http.StatusConflict, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось создать заказ"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().AddOrder(gomock.Any(), expectedOrder).Return(nil, storageErr)
			},
		},
	}

	for _, tt := range createOrderTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockOrderService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewOrderHandler(mock).CreateOrder, tt.in.body, nil)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestGetRestaurantOrders(t *testing.T) {
	storageErr := errors.New("storage failure")
	orders := []domain.Order{{ID: 11, RestaurantID: 17, Address: "Moscow", Status: domain.OrderStatusNew}}
	type in struct{ restaurantID string }
	type out struct {
		status int
		body   string
	}
	getRestaurantOrdersTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockOrderService)
	}{
		{
			name: "returns orders", in: in{restaurantID: "17"},
			out: out{status: http.StatusOK, body: jsonBody(t, []dto.OrderResponse{dto.NewOrderResponse(orders[0])})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().GetOrdersByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(orders, nil)
			},
		},
		{
			name: "returns empty orders", in: in{restaurantID: "17"},
			out: out{status: http.StatusOK, body: "[]"},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().GetOrdersByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(nil, nil)
			},
		},
		{
			name: "rejects invalid restaurant ID", in: in{restaurantID: "0"},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "service failure", in: in{restaurantID: "17"},
			out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().GetOrdersByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(nil, storageErr)
			},
		},
	}

	for _, tt := range getRestaurantOrdersTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockOrderService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewOrderHandler(mock).GetRestaurantOrders, "", gin.Params{{Key: "restaurant_id", Value: tt.in.restaurantID}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestChangeRestaurantOrderStatus(t *testing.T) {
	storageErr := errors.New("storage failure")
	type in struct {
		restaurantID string
		orderID      string
		body         string
	}
	type out struct {
		status int
		body   string
	}
	changeRestaurantOrderStatusTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockOrderService)
	}{
		{
			name: "changes status", in: in{restaurantID: "17", orderID: "11", body: `{"status":"accepted"}`},
			out: out{status: http.StatusNoContent},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().ChangeStatusForRestaurant(gomock.Any(), domain.RestaurantID(17), domain.OrderID(11), domain.OrderStatusAccepted).Return(nil)
			},
		},
		{
			name: "rejects invalid restaurant ID", in: in{restaurantID: "invalid", orderID: "11", body: `{"status":"accepted"}`},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "rejects invalid order ID", in: in{restaurantID: "17", orderID: "-1", body: `{"status":"accepted"}`},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор заказа"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "rejects malformed JSON", in: in{restaurantID: "17", orderID: "11", body: "{"},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные статуса заказа"})},
			setup: func(*mocks.MockOrderService) {},
		},
		{
			name: "order missing", in: in{restaurantID: "17", orderID: "11", body: `{"status":"accepted"}`},
			out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "заказ не найден"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().ChangeStatusForRestaurant(gomock.Any(), domain.RestaurantID(17), domain.OrderID(11), domain.OrderStatusAccepted).Return(service.ErrOrderNotFound)
			},
		},
		{
			name: "service failure", in: in{restaurantID: "17", orderID: "11", body: `{"status":"accepted"}`},
			out: out{status: http.StatusConflict, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось изменить статус заказа"})},
			setup: func(mock *mocks.MockOrderService) {
				mock.EXPECT().ChangeStatusForRestaurant(gomock.Any(), domain.RestaurantID(17), domain.OrderID(11), domain.OrderStatusAccepted).Return(storageErr)
			},
		},
	}

	for _, tt := range changeRestaurantOrderStatusTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockOrderService(gomock.NewController(t))
			tt.setup(mock)
			params := gin.Params{{Key: "restaurant_id", Value: tt.in.restaurantID}, {Key: "order_id", Value: tt.in.orderID}}
			recorder := callHandler(handler.NewOrderHandler(mock).ChangeRestaurantOrderStatus, tt.in.body, params)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}
