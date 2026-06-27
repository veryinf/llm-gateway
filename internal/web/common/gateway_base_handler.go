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
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

type GatewayBase struct {
	BaseHandler
	RouterService *service.RouterService
}

// IsDetailEnabled 判断是否记录详细请求/响应
func (h *GatewayBase) IsDetailEnabled() bool {
	return service.GetConfigString(model.ConfigKeyRequestLogDetail) == "true"
}

func (h *GatewayBase) HandleNonStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult) error {
	ctx := h.Context(c)

	resp, err := router.Adapter.AutoChat(c.Request().Context(), llmReq)
	log := h.buildRequestLog(ctx, router, llmReq, resp)
	if err != nil {
		log.ErrorMessage = err.Error()
		h.Store.RecordRequest(log)
		h.recordDetailIfEnabled(log, llmReq, resp, nil)
		return h.ErrorJSON(c, http.StatusBadGateway, err.Error())
	}
	if llmReq.APIType == model.APITypeAnthropic {
		resp, _ = resp.ToAnthropic()
	} else {
		resp, _ = resp.ToOpenAI()
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	_, copyErr := io.Copy(c.Response().Writer, resp.Response.Body)
	_ = resp.Response.Body.Close()

	if copyErr != nil {
		log.ErrorMessage = copyErr.Error()
		h.Store.RecordRequest(log)
		h.recordDetailIfEnabled(log, llmReq, resp, nil)
		return h.ErrorJSON(c, http.StatusBadGateway, copyErr.Error())
	}
	h.Store.RecordRequest(log)
	h.recordDetailIfEnabled(log, llmReq, resp, nil)
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

	chunkCollector := provider.NewChunkCollector(ctx.TraceID, h.IsDetailEnabled())
	log := h.buildRequestLog(ctx, router, llmReq, nil)
	chunkCh, errCh := router.Adapter.AutoStream(c.Request().Context(), llmReq)

	for chunk := range chunkCh {
		if llmReq.APIType == model.APITypeAnthropic {
			chunk, _ = chunk.ToAnthropic()
		} else {
			chunk, _ = chunk.ToOpenAI()
		}
		_, _ = fmt.Fprintf(c.Response().Writer, "data: %s\n\n", chunk.Raw)
		flusher.Flush()
		chunkCollector.Add(chunk)
		if chunk.Type == model.ChunkTypeUsage {
			extractUsageFromChunk(log, chunk)
		}
	}

	if err := <-errCh; err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		_, _ = fmt.Fprintf(c.Response().Writer, "data: %s\n\n", errData)
		flusher.Flush()
		log.ErrorMessage = err.Error()
	}

	log.Duration = time.Since(ctx.StartTime).Milliseconds()
	h.Store.RecordRequest(log)
	chunks := chunkCollector.Chunks()
	h.Store.RecordChunks(chunks)
	h.recordDetailIfEnabled(log, llmReq, nil, chunks)
	return nil
}

// ErrorJSON 返回 JSON 错误响应
func (h *GatewayBase) ErrorJSON(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]interface{}{"error": msg})
}

// recordDetailIfEnabled 当开启详细日志时，额外记录请求/响应详情
func (h *GatewayBase) recordDetailIfEnabled(log *model.RequestLog, req *provider.LLMRequest, resp *provider.LLMResponse, chunks []*model.RequestChunk) {
	if !h.IsDetailEnabled() {
		return
	}
	detail := &model.RequestDetail{
		TraceID: log.TraceID,
	}
	if req != nil && req.RawObject != nil {
		detail.RequestRaw = req.RawObject.Raw
		detail.Request = extractRequestText(req.RawObject)
	}
	if resp != nil && resp.RawObject != nil {
		detail.ResponseRaw = resp.RawObject.Raw
		extractResponse(detail, resp.RawObject)
	}
	if chunks != nil {
		detail.ResponseRaw = ""
		extractResponseFromChunks(detail, chunks)
	}
	h.Store.RecordDetail(log.TraceID, detail)
}

// extractSummary 从请求体中提取最后的用户问题作为摘要
func extractSummary(input *gjson.Result) string {
	if !input.IsObject() {
		return ""
	}
	messages := input.Get("messages")
	if !messages.Exists() || !messages.IsArray() {
		return ""
	}
	// 找到最后一条用户消息
	var lastUserContent string
	messages.ForEach(func(_, msg gjson.Result) bool {
		if msg.Get("role").String() == "user" {
			content := msg.Get("content")
			if content.IsArray() {
				content.ForEach(func(_, item gjson.Result) bool {
					if item.Get("type").String() == "text" {
						lastUserContent = item.Get("text").String()
					}
					return true
				})
			} else if content.Exists() {
				lastUserContent = content.String()
			}
		}
		return true
	})
	return core.TruncateStr(lastUserContent, 100)
}

// extractRequestText 从请求 JSON 提取纯文本（消息内容）
func extractRequestText(input *gjson.Result) string {
	if !input.IsObject() {
		return ""
	}
	messages := input.Get("messages")
	if !messages.Exists() || !messages.IsArray() {
		return ""
	}
	var parts []string
	messages.ForEach(func(_, msg gjson.Result) bool {
		role := msg.Get("role").String()
		content := msg.Get("content")
		if content.IsArray() {
			content.ForEach(func(_, item gjson.Result) bool {
				if item.Get("type").String() == "text" {
					parts = append(parts, item.Get("text").String())
				}
				return true
			})
		} else if content.Exists() && content.String() != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", role, content.String()))
		}
		return true
	})
	return strings.Join(parts, "\n")
}

// extractResponse 从响应 JSON 提取响应内容和推理内容
func extractResponse(detail *model.RequestDetail, input *gjson.Result) {
	if !input.IsObject() {
		return
	}
	// OpenAI 格式
	choices := input.Get("choices")
	if choices.Exists() && choices.IsArray() && len(choices.Array()) > 0 {
		msg := choices.Array()[0].Get("message")
		detail.Response = msg.Get("content").String()
		if reasoning := msg.Get("reasoning_content"); reasoning.Exists() && reasoning.String() != "" {
			detail.Reasoning = reasoning.String()
		}
		return
	}
	// Anthropic 格式
	content := input.Get("content")
	if content.Exists() && content.IsArray() && len(content.Array()) > 0 {
		detail.Response = content.Array()[0].Get("text").String()
	}
	thinking := input.Get("thinking")
	if thinking.Exists() && thinking.String() != "" {
		detail.Reasoning = thinking.String()
	}
}

// extractResponseFromChunks 从流式 chunks 提取响应内容和推理内容
func extractResponseFromChunks(detail *model.RequestDetail, chunks []*model.RequestChunk) {
	var responseParts, reasoningParts []string
	for _, chunk := range chunks {
		if !gjson.Valid(chunk.Data) {
			continue
		}
		result := gjson.Parse(chunk.Data)
		switch chunk.Type {
		case model.ChunkTypeMessage:
			// OpenAI: choices[0].delta.content
			if text := result.Get("choices.0.delta.content"); text.Exists() {
				responseParts = append(responseParts, text.String())
			}
			// Anthropic: delta.text
			if text := result.Get("delta.text"); text.Exists() {
				responseParts = append(responseParts, text.String())
			}
		case model.ChunkTypeReasoning:
			// OpenAI: choices[0].delta.reasoning_content
			if text := result.Get("choices.0.delta.reasoning_content"); text.Exists() {
				reasoningParts = append(reasoningParts, text.String())
			}
			// Anthropic: thinking
			if text := result.Get("delta.thinking"); text.Exists() {
				reasoningParts = append(reasoningParts, text.String())
			}
		}
	}
	detail.Response = strings.Join(responseParts, "")
	detail.Reasoning = strings.Join(reasoningParts, "")
	detail.ResponseRaw = ""
}

// buildRequestLog 构造 RequestLog（不含错误信息）
func (h *GatewayBase) buildRequestLog(c *LeContext, router *service.RouterResult, req *provider.LLMRequest, resp *provider.LLMResponse) *model.RequestLog {
	log := &model.RequestLog{
		TraceID:          c.TraceID,
		UserID:           c.AuthUser.UID,
		APIKeyID:         c.UserKey.KeyID,
		UserModel:        router.UserModelName,
		ProviderModel:    router.ProviderModelName,
		UserApiType:      req.APIType,
		ProviderApiType:  router.Adapter.ProviderAPIType(req.APIType),
		PassthroughLevel: router.Level,
		IsStream:         req.Stream,
		StatusCode:       http.StatusOK,
		Duration:         time.Since(c.StartTime).Milliseconds(),
		IPAddress:        c.RealIP(),
		UserAgent:        req.Request.UserAgent(),
		Summary:          extractSummary(req.RawObject),
		IsDetail:         h.IsDetailEnabled(),
		CreatedAt:        time.Now(),
	}
	if resp != nil {
		log.ProviderApiType = resp.APIType
		if resp.Converted {
			// 转换后的 APIType 是输出格式，ProviderApiType 应是原始 Provider 的类型
			log.ProviderApiType = lo.If(resp.APIType == model.APITypeOpenAI, model.APITypeAnthropic).Else(model.APITypeOpenAI)
		}
		log.StatusCode = resp.StatusCode
		log.ResponseModel = resp.RawObject.Get("model").String()
		extractUsage(log, resp)
	}
	return log
}

// extractUsage 从响应中提取 token 用量（自动识别 Anthropic / OpenAI 格式）
func extractUsage(log *model.RequestLog, resp *provider.LLMResponse) {
	usage := resp.RawObject.Get("usage")
	if !usage.Exists() {
		return
	}
	if usage.Get("input_tokens").Exists() {
		log.PromptTokens = int(usage.Get("input_tokens").Int())
		log.CompletionTokens = int(usage.Get("output_tokens").Int())
		log.CachedTokens = int(usage.Get("cache_read_input_tokens").Int())
		log.TotalTokens = log.PromptTokens + log.CompletionTokens
	} else {
		log.PromptTokens = int(usage.Get("prompt_tokens").Int())
		log.CompletionTokens = int(usage.Get("completion_tokens").Int())
		log.TotalTokens = int(usage.Get("total_tokens").Int())
		log.CachedTokens = int(usage.Get("prompt_tokens_details.cached_tokens").Int())
		log.ReasoningTokens = int(usage.Get("completion_tokens_details.reasoning_tokens").Int())
	}
}

// extractUsageFromChunk 从 chunk 中提取 usage 信息更新到 log
func extractUsageFromChunk(log *model.RequestLog, chunk *provider.LLMResponseChunk) {
	if chunk.Type != model.ChunkTypeUsage {
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
		log.CachedTokens = int(usage.Get("cache_read_input_tokens").Int())
		log.TotalTokens = log.PromptTokens + log.CompletionTokens
	} else {
		log.PromptTokens = int(usage.Get("prompt_tokens").Int())
		log.CompletionTokens = int(usage.Get("completion_tokens").Int())
		log.TotalTokens = int(usage.Get("total_tokens").Int())
		log.CachedTokens = int(usage.Get("prompt_tokens_details.cached_tokens").Int())
		log.ReasoningTokens = int(usage.Get("completion_tokens_details.reasoning_tokens").Int())
	}
}
