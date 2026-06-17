package auth

import "errors"

var ErrWrongLoginOrPassword error = errors.New("Неверный логин или пароль")
var ErrNoAuthToken error = errors.New("Отсутствует токен авторизации")
var ErrAuthTokenNotValidOrExpired error = errors.New("Неверный токен авторизации или истек срок действия")
