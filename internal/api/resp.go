package api

import (
	"net/http"
	"shortpress-server/internal/common"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	Code int         `json:"code"`
	Info string      `json:"info"`
	Data interface{} `json:"data"`
}

func HandleSuccess(ctx *gin.Context, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	errorCodeMap := common.GetErrorCodeMap()
	resp := Response{Code: errorCodeMap[common.ErrSuccess], Info: common.ErrSuccess.Error(), Data: data}
	if _, ok := errorCodeMap[common.ErrSuccess]; !ok {
		resp = Response{Code: 0, Info: "", Data: data}
	}
	ctx.JSON(http.StatusOK, resp)
}

func HandleErrorWithHttpCode(ctx *gin.Context, httpCode int, err error, data interface{}) {
	if data == nil {
		data = map[string]string{}
	}
	log.AddNotice(ctx, "error", err.Error())
	errorCodeMap := common.GetErrorCodeMap()
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		ctx.JSON(httpCode, Response{
			Code: errorCodeMap[common.ErrUnknown],
			Info: validationErrs.Error(), // Format the validation errors nicely
			Data: data,
		})
		return
	}
	resp := Response{Code: errorCodeMap[err], Info: err.Error(), Data: data}
	if _, ok := errorCodeMap[err]; !ok {
		resp = Response{Code: httpCode, Info: err.Error(), Data: data}
	}
	ctx.JSON(httpCode, resp)
}

func HandleError(ctx *gin.Context, err error, data interface{}) {
	if data == nil {
		data = map[string]string{}
	}
	log.AddNotice(ctx, "error", err.Error())
	errorCodeMap := common.GetErrorCodeMap()
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		ctx.JSON(http.StatusBadRequest, Response{
			Code: errorCodeMap[common.ErrUnknown],
			Info: validationErrs.Error(), // Format the validation errors nicely
			Data: data,
		})
		return
	}

	resp := &Response{Code: errorCodeMap[common.ErrUnknown], Info: err.Error(), Data: data}
	httpCode := http.StatusInternalServerError
	if _, ok := errorCodeMap[err]; ok {
		httpCode = http.StatusOK
		resp = &Response{Code: errorCodeMap[err], Info: err.Error(), Data: data}
	}
	ctx.JSON(httpCode, resp)
}
