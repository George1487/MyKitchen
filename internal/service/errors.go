package service

import "errors"

var (
	ErrRestaurantNotFound = errors.New("ресторан не найден")
	ErrMenuItemNotFound   = errors.New("позиция меню не найдена")
	ErrOrderNotFound      = errors.New("заказ не найден")
)
