package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Status  string      `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, code int, data interface{}) {
	c.JSON(code, APIResponse{
		Status: "success",
		Data:   data,
	})
}

func SuccessWithMessage(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, APIResponse{
		Status:  "error",
		Message: message,
	})
}