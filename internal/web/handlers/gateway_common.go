package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type GatewayBase struct {
	common.BaseHandler
	RouterSvc *service.RouterService
	LogSvc    *service.RequestLogService
}

// gatewayContext 保存单次请求的公共上下文
type gatewayContext struct {
	TraceID   string
	UserID    uint
	APIKeyID  uint
	StartTime time.Time
	ReqBytes  []byte
	Flush     http.Flusher
}

// Latency 计算从 StartTime 到现在的延迟毫秒数
func (gw *gatewayContext) Latency() int64 {
	return time.Since(gw.StartTime).Milliseconds()
}

// prepareRequest 初始化请求上下文：生成 traceID、读取 body、解析 JSON、提取用户信息
func (h *GatewayBase) prepareRequest(c echo.Context, req interface{}) (*gatewayContext, error) {
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)

	rawBody, _ := io.ReadAll(c.Request().Body)
	c.Request().Body = io.NopCloser(bytes.NewReader(rawBody))

	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(req); err != nil {
		return nil, fmt.Errorf("invalid request: %v", err)
	}

	uid, kid := extractUserInfo(c)

	return &gatewayContext{
		TraceID:   traceID,
		UserID:    uid,
		APIKeyID:  kid,
		StartTime: time.Now(),
		ReqBytes:  rawBody,
	}, nil
}

// prepareStream 设置 SSE 头部并获取 flusher
func (h *GatewayBase) prepareStream(c echo.Context) (http.Flusher, error) {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	return flusher, nil
}

// extractUserInfo 从 LeContext 提取用户 ID 和 API Key ID
func extractUserInfo(c echo.Context) (uid, kid uint) {
	cc, ok := c.(*common.LeContext)
	if !ok {
		return
	}
	if cc.AuthUser != nil {
		uid = cc.AuthUser.UID
	}
	if cc.UserKey != nil {
		kid = cc.UserKey.KeyID
	}
	return
}

// errorJSON 返回 JSON 错误响应
func (h *GatewayBase) errorJSON(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]interface{}{"error": msg})
}

// resolveProvider 委托给 RouterService
// 返回: adapter, providerModelName, passthroughLevel, error
func (h *GatewayBase) resolveProvider(modelName string) (*provider.Adapter, string, string, error) {
	return h.RouterSvc.ResolveProvider(modelName)
}

// recordRequest 委托给 RequestLogService
func (h *GatewayBase) recordRequest(traceID string, userID, apiKeyID uint,
	userModel, providerModel, userApiType, providerApiType, passthroughLevel string, isStream bool,
	promptTokens, completionTokens, totalTokens, cachedTokens int,
	statusCode int, errMsg string, duration int64, c echo.Context,
	reqBytes, respBytes []byte, chunks []*model.RequestChunk) {

	h.LogSvc.RecordRequest(traceID, userID, apiKeyID,
		userModel, providerModel, userApiType, providerApiType, passthroughLevel, isStream,
		promptTokens, completionTokens, totalTokens, cachedTokens,
		statusCode, errMsg, duration,
		c.RealIP(), c.Request().Header.Get("User-Agent"),
		reqBytes, respBytes, chunks)
}

// newChunkCollector 委托给 RequestLogService
func (h *GatewayBase) newChunkCollector(traceID string) *service.StreamChunkCollector {
	return h.LogSvc.NewChunkCollector(traceID)
}
