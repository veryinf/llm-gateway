package response

import (
	"net/http"

	"llm-gateway/pkg/apierror"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "ok",
		Data: data,
	})
}

func SuccessPage(c echo.Context, list interface{}, total int64, page, pageSize int) error {
	return c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "ok",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

func Error(c echo.Context, err error) error {
	if apiErr, ok := err.(*apierror.APIError); ok {
		return c.JSON(apiErr.HTTP, Response{
			Code: int(apiErr.Code),
			Msg:  apiErr.Message,
		})
	}

	return c.JSON(http.StatusInternalServerError, Response{
		Code: int(apierror.CodeInternalError),
		Msg:  err.Error(),
	})
}
