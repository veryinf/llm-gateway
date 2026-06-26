package common

import (
	"encoding/json"
	"fmt"
	"io"
	"llm-gateway/internal/core"
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

func (h *GatewayBase) HandleNonStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult) error {
	ctx := h.Context(c)

	resp, err := router.Adapter.AutoChat(c.Request().Context(), llmReq)
	log := h.BuildRequestLog(ctx, router, llmReq, resp)
	if err != nil {
		log.ErrorMessage = err.Error()
		h.RequestLogService.RecordRequest(log)
		h.RecordDetailIfEnabled(log, llmReq, resp)
		return h.ErrorJSON(c, http.StatusBadGateway, err.Error())
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	_, copyErr := io.Copy(c.Response().Writer, resp.Response.Body)
	_ = resp.Response.Body.Close()

	if copyErr != nil {
		log.ErrorMessage = copyErr.Error()
		h.RequestLogService.RecordRequest(log)
		h.RecordDetailIfEnabled(log, llmReq, resp)
		return h.ErrorJSON(c, http.StatusBadGateway, copyErr.Error())
	}
	h.RequestLogService.RecordRequest(log)
	h.RecordDetailIfEnabled(log, llmReq, resp)
	return nil
}

func (h *GatewayBase) HandleStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return h.ErrorJSON(c, http.StatusInternalServerError, "streaming not supported")
	}

	ctx := h.Context(c)

	chunkCollector := provider.NewChunkCollector(ctx.TraceID, h.RequestLogService.IsDetailEnabled())
	log := h.BuildRequestLog(ctx, router, llmReq, nil)
	chunkCh, errCh := router.Adapter.ChatCompletionStream(c.Request().Context(), llmReq)

	for chunk := range chunkCh {
		if llmReq.APIType == model.APITypeAnthropic {
			chunk, _ = chunk.ToAnthropic()
		} else {
			chunk, _ = chunk.ToOpenAI()
		}
		_, _ = fmt.Fprintf(c.Response().Writer, "data: %s\n\n", chunk.Raw)
		flusher.Flush()
		chunkCollector.Add(chunk.RawData)
		if chunk.Type == provider.ChunkTypeUsage {
			h.RequestUsageFromChunk(log, chunk)
		}
	}

	if err := <-errCh; err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		_, _ = fmt.Fprintf(c.Response().Writer, "data: %s\n\n", errData)
		flusher.Flush()
		log.ErrorMessage = err.Error()
	}

	h.RequestLogService.RecordRequest(log)
	h.RequestLogService.RecordChunks(chunkCollector.Chunks())
	return nil
}

// ErrorJSON 返回 JSON 错误响应
func (h *GatewayBase) ErrorJSON(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]interface{}{"error": msg})
}

// RecordDetailIfEnabled 当开启详细日志时，额外记录请求/响应详情
func (h *GatewayBase) RecordDetailIfEnabled(log *model.RequestLog, req *provider.LLMRequest, resp *provider.LLMResponse) {
	if !h.RequestLogService.IsDetailEnabled() {
		return
	}
	h.RequestLogService.RecordDetail(log.TraceID, req.RawObject.Raw, lo.If(resp != nil, resp.RawObject.Raw).Else(""))
}

// BuildRequestLog 构造 RequestLog（不含错误信息）
func (h *GatewayBase) BuildRequestLog(c *LeContext, router *service.RouterResult, req *provider.LLMRequest, resp *provider.LLMResponse) *model.RequestLog {
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
		Summary:          extractSummary(req.RawObject),
	}
	if resp != nil {
		log.ProviderApiType = resp.APIType
		if resp.Converted {
			// 转换后的 APIType 是输出格式，ProviderApiType 应是原始 Provider 的类型
			log.ProviderApiType = lo.If(resp.APIType == model.APITypeOpenAI, model.APITypeAnthropic).Else(model.APITypeOpenAI)
		}
		log.StatusCode = resp.StatusCode
		log.ResponseModel = resp.RawObject.Get("model").String()
		extractTokenUsage(log, resp)
	}
	return log
}

// extractTokenUsage 从响应中提取 token 用量（自动识别 Anthropic / OpenAI 格式）
func extractTokenUsage(log *model.RequestLog, resp *provider.LLMResponse) {
	usage := resp.RawObject.Get("usage")
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
				return core.TruncateStr(inputTexts[len(inputTexts)-1].Get("text").String(), 100)
			}
		} else {
			return core.TruncateStr(inputContent.String(), 100)
		}
	}
	return ""
}

// RequestUsageFromChunk 从 chunk 中提取 usage 信息更新到 log
func (h *GatewayBase) RequestUsageFromChunk(log *model.RequestLog, chunk *provider.LLMResponseChunk) {
	if chunk.Type != provider.ChunkTypeUsage {
		return
	}
	if chunk.RawObject == nil {
		return
	}
	usage := chunk.RawObject.Get("usage")
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
