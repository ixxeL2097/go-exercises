package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func HTTPRes(c *gin.Context, httpCode int, msg string, data interface{}) {
	if httpCode >= 500 {
		log.Error(msg)
	} else if httpCode >= 400 {
		log.Warn(msg)
	} else {
		log.Info(msg)
	}
	c.JSON(httpCode, Response{
		Code: httpCode,
		Msg:  msg,
		Data: data,
	})
}

func Success(c *gin.Context, msg string, data interface{}) {
	HTTPRes(c, http.StatusOK, msg, data)
}

func Error(c *gin.Context, httpCode int, msg string) {
	HTTPRes(c, httpCode, msg, nil)
}
