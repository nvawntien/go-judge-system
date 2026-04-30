package handler

import userhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	GetMe      *userhandler.GetMeHandler
	GetProfile *userhandler.GetProfileHandler
}

func NewUserHandler(getMe *userhandler.GetMeHandler, getProfile *userhandler.GetProfileHandler) *UserHandler {
	return &UserHandler{
		GetMe:      getMe,
		GetProfile: getProfile,
	}
}
