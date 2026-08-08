package httpx

import "github.com/gin-gonic/gin"

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

func Error(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorPayload{Code: code, Message: msg}})
}
