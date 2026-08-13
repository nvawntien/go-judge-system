package handler

import userhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/user"

type UserHandler struct {
	GetMe             *userhandler.GetMeHandler
	GetProfile        *userhandler.GetProfileHandler
	SearchPublicUsers *userhandler.SearchPublicUsersHandler
	UpdateProfile     *userhandler.UpdateProfileHandler
	UploadAvatar      *userhandler.UploadAvatarHandler
}

func NewUserHandler(
	getMe *userhandler.GetMeHandler,
	getProfile *userhandler.GetProfileHandler,
	searchPublicUsers *userhandler.SearchPublicUsersHandler,
	updateProfile *userhandler.UpdateProfileHandler,
	uploadAvatar *userhandler.UploadAvatarHandler,
) *UserHandler {
	return &UserHandler{
		GetMe:             getMe,
		GetProfile:        getProfile,
		SearchPublicUsers: searchPublicUsers,
		UpdateProfile:     updateProfile,
		UploadAvatar:      uploadAvatar,
	}
}
