package handler

import userhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	GetMe *userhandler.GetMeHandler
}

func NewUserHandler(getMe *userhandler.GetMeHandler) *UserHandler {
	return &UserHandler{GetMe: getMe}
}
