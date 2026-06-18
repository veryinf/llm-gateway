package handlers

import (
	"fmt"
	"sync/atomic"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/web/common"

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
	// 1. 查询 UserModel 表，找到 user_model_id（根据 modelName）
	var userModel model.UserModel
	if err := h.DB.Where("name = ? AND is_active = ?", modelName, true).First(&userModel).Error; err != nil {
		return nil, fmt.Errorf("model %q not found", modelName)
	}

	// 2. 查询 UserModelRouter 表，找到 provider_model_id（按优先级排序）
	var routerEntry model.UserModelRouter
	if err := h.DB.Where("user_model_id = ?", userModel.UserModelID).
		Order("priority ASC, router_id ASC").
		First(&routerEntry).Error; err != nil {
		return nil, fmt.Errorf("no router entry for model %q", modelName)
	}

	// 3. 查询 ProviderModel 表，获取 provider_id 和 name
	var providerModel model.ProviderModel
	if err := h.DB.Where("model_id = ? AND is_active = ?", routerEntry.ProviderModelID, true).
		First(&providerModel).Error; err != nil {
		return nil, fmt.Errorf("provider model %d not found", routerEntry.ProviderModelID)
	}

	// 4. 查询 Provider 表，获取 provider 配置
	var prov model.Provider
	if err := h.DB.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).
		First(&prov).Error; err != nil {
		return nil, fmt.Errorf("provider %d not found", providerModel.ProviderID)
	}

	// 5. 根据配置动态创建 provider 适配器
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

// getAllModels 获取所有已注册的模型列表
func (h *GatewayBase) getAllModels() []provider.ModelInfo {
	var userModels []model.UserModel
	if err := h.DB.Where("is_active = ?", true).Find(&userModels).Error; err != nil {
		return nil
	}

	var models []provider.ModelInfo
	for _, um := range userModels {
		// 检查是否有路由配置
		var count int64
		h.DB.Model(&model.UserModelRouter{}).Where("user_model_id = ?", um.UserModelID).Count(&count)
		if count > 0 {
			models = append(models, provider.ModelInfo{
				ID:     um.Name,
				Object: "model",
			})
		}
	}

	return models
}
