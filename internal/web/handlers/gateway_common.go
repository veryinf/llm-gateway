package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/web/common"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StatsRecorder interface {
	Record(log *model.RequestLog)
}

type ChunkRecorder interface {
	Record(chunk *model.RequestChunk)
}

type GatewayBase struct {
	common.BaseHandler
	StatsSvc  StatsRecorder
	ChunkSvc  ChunkRecorder
	LogDetail *atomic.Bool
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

// streamChunkCollector 收集流式响应的 chunks
type streamChunkCollector struct {
	traceID   string
	logDetail bool
	chunks    []*model.RequestChunk
	index     int
}

func (h *GatewayBase) newChunkCollector(traceID string) *streamChunkCollector {
	return &streamChunkCollector{
		traceID:   traceID,
		logDetail: h.LogDetail.Load(),
	}
}

func (sc *streamChunkCollector) Add(data []byte) {
	if !sc.logDetail {
		return
	}
	sc.chunks = append(sc.chunks, &model.RequestChunk{
		TraceID:    sc.traceID,
		ChunkIndex: sc.index,
		ChunkData:  string(data),
		CreatedAt:  time.Now(),
	})
	sc.index++
}

func (sc *streamChunkCollector) Chunks() []*model.RequestChunk {
	if !sc.logDetail {
		return nil
	}
	return sc.chunks
}

func (h *GatewayBase) recordRequest(traceID string, userID, apiKeyID uint,
	providerName, modelName string, isStream bool,
	promptTokens, completionTokens, totalTokens int,
	statusCode int, errMsg string, latencyMs int64, c echo.Context,
	reqBytes, respBytes []byte, chunks []*model.RequestChunk) {

	if h.StatsSvc == nil {
		return
	}

	logDetail := h.LogDetail.Load()
	var requestBody, responseBody string
	var isDetail bool
	if logDetail {
		isDetail = true
		requestBody = truncateStr(string(reqBytes), 65536)
		if respBytes != nil {
			responseBody = truncateStr(string(respBytes), 65536)
		}
	}

	h.StatsSvc.Record(&model.RequestLog{
		TraceID:          traceID,
		UserID:           userID,
		APIKeyID:         apiKeyID,
		ModelName:        modelName,
		IsStream:         isStream,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		RequestBody:      requestBody,
		ResponseBody:     responseBody,
		IsDetail:         isDetail,
		StatusCode:       statusCode,
		ErrorMessage:     truncateStr(errMsg, 4096),
		LatencyMs:        latencyMs,
		Cost:             0,
		IPAddress:        c.RealIP(),
		UserAgent:        truncateStr(c.Request().Header.Get("User-Agent"), 512),
		CreatedAt:        time.Now(),
	})

	if logDetail && len(chunks) > 0 && h.ChunkSvc != nil {
		for _, chunk := range chunks {
			h.ChunkSvc.Record(chunk)
		}
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// resolveProvider 根据模型名称解析到对应的 Provider
func (h *GatewayBase) resolveProvider(modelName string) (provider.LLMProvider, error) {
	var userModel model.UserModel
	if err := h.DB.Where("name = ? AND is_active = ?", modelName, true).First(&userModel).Error; err != nil {
		// UserModel 找不到，检查透传级别
		level := h.getPassthroughLevel()
		if level == "none" {
			return nil, fmt.Errorf("model %q not found", modelName)
		}
		return h.resolveProviderByModelName(modelName, level)
	}

	var routerEntry model.UserModelRouter
	if err := h.DB.Where("user_model_id = ?", userModel.UserModelID).
		Order("priority ASC, router_id ASC").
		First(&routerEntry).Error; err != nil {
		return nil, fmt.Errorf("no router entry for model %q", modelName)
	}

	var providerModel model.ProviderModel
	if err := h.DB.Where("model_id = ? AND is_active = ?", routerEntry.ProviderModelID, true).
		First(&providerModel).Error; err != nil {
		return nil, fmt.Errorf("provider model %d not found", routerEntry.ProviderModelID)
	}

	var prov model.Provider
	if err := h.DB.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).
		First(&prov).Error; err != nil {
		return nil, fmt.Errorf("provider %d not found", providerModel.ProviderID)
	}

	return createProviderAdapter(prov)
}

// getPassthroughLevel 获取透传级别配置
func (h *GatewayBase) getPassthroughLevel() string {
	var config model.Config
	if err := h.DB.Where("key = ?", model.ConfigKeyRouterPassthrough).First(&config).Error; err != nil {
		return "none"
	}
	switch config.Value {
	case "user", "provider":
		return config.Value
	default:
		return "none"
	}
}

// resolveProviderByModelName 透传：直接匹配 ProviderModel
func (h *GatewayBase) resolveProviderByModelName(modelName, level string) (provider.LLMProvider, error) {
	var providerModel model.ProviderModel
	if err := h.DB.Where("name = ? AND is_active = ?", modelName, true).First(&providerModel).Error; err != nil {
		if level == "provider" {
			return h.resolveProviderByDefault(modelName)
		}
		return nil, fmt.Errorf("model %q not found in provider models", modelName)
	}

	var prov model.Provider
	if err := h.DB.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).First(&prov).Error; err != nil {
		return nil, fmt.Errorf("provider %d not found", providerModel.ProviderID)
	}

	return createProviderAdapter(prov)
}

// resolveProviderByDefault 二级透传：使用 default Provider
func (h *GatewayBase) resolveProviderByDefault(modelName string) (provider.LLMProvider, error) {
	var prov model.Provider
	if err := h.DB.Where("is_default = ? AND is_active = ?", true, true).First(&prov).Error; err != nil {
		return nil, fmt.Errorf("no default provider configured for model %q", modelName)
	}

	return createProviderAdapter(prov)
}

// createProviderAdapter 根据 provider 配置创建适配器
func createProviderAdapter(p model.Provider) (provider.LLMProvider, error) {
	if p.SupportOpenAI {
		url := p.OpenAIBaseURL
		if url == "" {
			url = p.BaseURL + "/v1"
		}
		return provider.NewOpenAICompatibleProvider(p.Title, url, p.APIKey), nil
	}

	if p.SupportAnthropic {
		url := p.AnthropicBaseURL
		if url == "" {
			url = p.BaseURL + "/anthropic/v1"
		}
		return provider.NewAnthropicProvider(p.Title, url, p.APIKey), nil
	}

	return nil, fmt.Errorf("provider %q has no supported API types", p.Title)
}
