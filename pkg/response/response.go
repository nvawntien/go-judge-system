package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Data   any    `json:"data,omitempty"`
}

func Success(c *gin.Context, code int, data any) {
	c.JSON(GetHTTPStatus(code), APIResponse{
		Status: "success",
		Code:   code,
		Data:   data,
	})
}

func SuccessWithMessage(c *gin.Context, code int, msg string, data any) {
	c.JSON(GetHTTPStatus(code), APIResponse{
		Status: "success",
		Code:   code,
		Msg:    msg,
		Data:   data,
	})
}

func Error(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(GetHTTPStatus(code), APIResponse{
		Status: "error",
		Code:   code,
		Msg:    msg,
	})
}