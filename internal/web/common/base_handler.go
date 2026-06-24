package common

import (
	"io"
	"log/slog"
	"net/http"

	"llm-gateway/internal/core"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

type BaseHandler struct {
	DB           *gorm.DB
	Store        *sqlx.DB
	Config       *core.Config
	TokenManager *TokenManager
}

func (h *BaseHandler) Context(c echo.Context) *LeContext {
	return c.(*LeContext)
}

func (h *BaseHandler) Success() *ResponseStruct {
	return h.Error(0, "ok")
}

func (h *BaseHandler) Error(code int, msg string) *ResponseStruct {
	return &ResponseStruct{
		ErrCode: code,
		ErrMsg:  msg,
	}
}

func (h *BaseHandler) Pagination(input *Pagination, query *gorm.DB, dataSet any, count *int64) error {
	if input.Index == 0 {
		input.Index = 1
	}
	if input.Size == 0 {
		input.Size = 10
	}
	input.Offset = (input.Index - 1) * input.Size
	if err := query.Count(count).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	if err := query.Offset(input.Offset).Limit(input.Size).Find(dataSet).Error; err != nil {
		return h.Error(-20, err.Error())
	}
	return nil
}

func (h *BaseHandler) GetJSON(c echo.Context) (json gjson.Result, err error) {
	body, err := io.ReadAll(c.Request().Body)
	if err == nil {
		json = gjson.ParseBytes(body)
	} else {
		err = h.Error(-20, "invalid json")
	}
	return
}

func (h *BaseHandler) UseSSE(c echo.Context) (flusher func(), writer func(data string), done func()) {
	w := c.Response()
	w.Header().Set(echo.HeaderContentType, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher = func() {
		err := http.NewResponseController(w).Flush()
		if err != nil {
			slog.Warn("flush error", "error", err)
		}
	}
	flusher()
	writer = func(data string) {
		_, err := w.Write([]byte(data))
		if err != nil {
			slog.Warn("write error", "error", err)
		}
	}
	done = func() {
		<-c.Request().Context().Done()
	}
	return
}
