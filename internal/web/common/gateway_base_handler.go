package common

import (
	"fmt"
	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/service"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

type GatewayBase struct {
	BaseHandler
	RouterService     *service.RouterService
	RequestLogService *service.RequestLogService
}

// PrepareStream 设置 SSE 头部并获取 flusher
func (h *GatewayBase) PrepareStream(c echo.Context) (http.Flusher, error) {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	return flusher, nil
}

// ErrorJSON 返回 JSON 错误响应
func (h *GatewayBase) ErrorJSON(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]interface{}{"error": msg})
}

// RecordRequest 记录请求日志
// resp 非 nil 时使用其状态码与 API 类型；流式响应时 resp 为 nil
func (h *GatewayBase) RecordRequest(c *LeContext, router *service.RouterResult, req *provider.LLMRequest, resp *provider.LLMResponse) {
	log := h.BuildLog(c, router, req, resp)
	h.RequestLogService.RecordRequest(log)
	h.recordDetailIfEnabled(log, req, resp)
}

// RecordRequestWithError 记录请求日志（带错误信息）
// 错误场景专用：覆盖默认 200 状态码并写入错误消息
func (h *GatewayBase) RecordRequestWithError(c *LeContext, router *service.RouterResult, req *provider.LLMRequest, resp *provider.LLMResponse, code int, message string) {
	log := h.BuildLog(c, router, req, resp)
	log.StatusCode = code
	log.ErrorMessage = message
	h.RequestLogService.RecordRequest(log)
	h.recordDetailIfEnabled(log, req, resp)
}

// RecordChunks 记录流式响应 chunks
func (h *GatewayBase) RecordChunks(chunks []*model.RequestChunk) {
	h.RequestLogService.RecordChunks(chunks)
}

// NewChunkCollector 委托给 RequestLogService
func (h *GatewayBase) NewChunkCollector(traceID string) *service.StreamChunkCollector {
	return h.RequestLogService.NewChunkCollector(traceID)
}

// BuildLog 构造 RequestLog（不含错误信息）
func (h *GatewayBase) BuildLog(c *LeContext, router *service.RouterResult, req *provider.LLMRequest, resp *provider.LLMResponse) *model.RequestLog {
	log := &model.RequestLog{
		TraceID:          c.TraceID,
		UserID:           c.AuthUser.UID,
		APIKeyID:         c.UserKey.KeyID,
		UserModel:        router.UserModelName,
		ProviderModel:    router.ProviderModelName,
		UserApiType:      req.APIType,
		ProviderApiType:  router.ProviderAPIType(),
		PassthroughLevel: router.Level,
		IsStream:         req.Stream,
		StatusCode:       http.StatusOK,
		Duration:         time.Since(c.StartTime).Milliseconds(),
		IPAddress:        c.RealIP(),
		UserAgent:        req.Request.UserAgent(),
		Summary:          extractSummary(req.BodyRaw),
	}
	if resp != nil {
		log.ProviderApiType = resp.APIType
		if resp.Converted {
			// 转换后的 APIType 是输出格式，ProviderApiType 应是原始 Provider 的类型
			log.ProviderApiType = lo.If(resp.APIType == model.APITypeOpenAI, model.APITypeAnthropic).Else(model.APITypeOpenAI)
		}
		log.StatusCode = resp.StatusCode
		log.ResponseModel = resp.BodyRaw.Get("model").String()
		extractTokenUsage(log, resp)
	}
	return log
}

// extractTokenUsage 从响应中提取 token 用量（自动识别 Anthropic / OpenAI 格式）
func extractTokenUsage(log *model.RequestLog, resp *provider.LLMResponse) {
	usage := resp.BodyRaw.Get("usage")
	if !usage.Exists() {
		return
	}
	if usage.Get("input_tokens").Exists() {
		log.PromptTokens = int(usage.Get("input_tokens").Int())
		log.CompletionTokens = int(usage.Get("output_tokens").Int())
		log.TotalTokens = log.PromptTokens + log.CompletionTokens
	} else {
		log.PromptTokens = int(usage.Get("prompt_tokens").Int())
		log.CompletionTokens = int(usage.Get("completion_tokens").Int())
		log.TotalTokens = int(usage.Get("total_tokens").Int())
		log.CachedTokens = int(usage.Get("prompt_tokens_details.cached_tokens").Int())
		log.ReasoningTokens = int(usage.Get("completion_tokens_details.reasoning_tokens").Int())
	}
}

// recordDetailIfEnabled 当开启详细日志时，额外记录请求/响应详情
func (h *GatewayBase) recordDetailIfEnabled(log *model.RequestLog, req *provider.LLMRequest, resp *provider.LLMResponse) {
	if !h.RequestLogService.IsDetailEnabled() {
		return
	}
	var reqBytes, respBytes []byte
	if req != nil {
		reqBytes = []byte(req.BodyRaw.Raw)
	}
	if resp != nil {
		respBytes = []byte(resp.BodyRaw.Raw)
	}
	h.RequestLogService.RecordDetail(log.TraceID, reqBytes, respBytes)
}

// extractSummary 从请求体中提取最后的用户问题作为摘要
func extractSummary(input *gjson.Result) string {
	if !input.IsObject() {
		return ""
	}
	inputMessages := input.Get("messages")
	if !inputMessages.Exists() || !inputMessages.IsArray() {
		return ""
	}
	messages := lo.Filter(inputMessages.Array(), func(msg gjson.Result, _ int) bool {
		return msg.Get("role").String() == "user"
	})
	if len(messages) > 0 {
		inputContent := messages[len(messages)-1].Get("content")
		if inputContent.IsArray() {
			inputTexts := lo.Filter(inputContent.Array(), func(content gjson.Result, _ int) bool {
				return content.Get("type").Str == "text"
			})
			if len(inputTexts) > 0 {
				return truncateStr(inputTexts[len(inputTexts)-1].Get("text").String(), 100)
			}
		} else {
			return truncateStr(inputContent.String(), 100)
		}
	}
	return ""
}

// truncateStr 截断字符串到指定最大长度（与 service.TruncateStr 等价，避免包间依赖）
func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
