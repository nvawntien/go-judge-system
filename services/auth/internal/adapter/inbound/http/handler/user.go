package handler

import userhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	GetMe         *userhandler.GetMeHandler
	GetProfile    *userhandler.GetProfileHandler
	UpdateProfile *userhandler.UpdateProfileHandler
}

func NewUserHandler(
	getMe *userhandler.GetMeHandler,
	getProfile *userhandler.GetProfileHandler,
	updateProfile *userhandler.UpdateProfileHandler,
) *UserHandler {
	return &UserHandler{
		GetMe:         getMe,
		GetProfile:    getProfile,
		UpdateProfile: updateProfile,
	}
}
