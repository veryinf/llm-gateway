package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"llm-gateway/internal/model"

	"github.com/tidwall/gjson"
)

// LLMRequest LLM 请求封装
type LLMRequest struct {
	APIType   model.LLMAPIType // API 类型：openai / anthropic
	Request   *http.Request    // 原始 HTTP 请求（用于转发，可能被转换）
	Model     string           // 模型名称（用于路由判断）
	Stream    bool             // 是否流式（用于路由判断）
	RawObject *gjson.Result    // 原始请求体对象（用于日志记录）
}

// NewLLMRequest 从 *http.Request 解析 LLM 请求
// apiType: model.APITypeOpenAI 或 model.APITypeAnthropic
// 使用 TeeReader 复制 body，原始请求不被消耗
func NewLLMRequest(req *http.Request, apiType model.LLMAPIType) (*LLMRequest, error) {
	// 使用 TeeReader 读取 body 并同时复制一份
	var bodyBuf bytes.Buffer
	bodyBytes, err := io.ReadAll(io.TeeReader(req.Body, &bodyBuf))
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()

	// 重建原始请求的 body（保留原始流不被消耗）
	req.Body = io.NopCloser(&bodyBuf)
	// 使用 gjson 解析必要字段
	if !gjson.ValidBytes(bodyBytes) {
		return nil, fmt.Errorf("invalid request body")
	}
	result := gjson.ParseBytes(bodyBytes)
	return &LLMRequest{
		APIType:   apiType,
		Request:   req,
		Model:     result.Get("model").String(),
		Stream:    result.Get("stream").Bool(),
		RawObject: &result,
	}, nil
}

// ToAnthropic 将请求转为 Anthropic 格式
func (r *LLMRequest) ToAnthropic() (*LLMRequest, error) {
	if r.APIType == model.APITypeAnthropic {
		return r, nil
	}
	newRequest, err := RequestOpenAIToAnthropic(r.Request)
	if err != nil {
		return nil, err
	}
	return &LLMRequest{
		Request:   newRequest,
		APIType:   model.APITypeAnthropic,
		Model:     r.Model,
		Stream:    r.Stream,
		RawObject: r.RawObject,
	}, nil
}

// ToOpenAI 将请求转为 OpenAI 格式
func (r *LLMRequest) ToOpenAI() (*LLMRequest, error) {
	if r.APIType == model.APITypeOpenAI {
		return r, nil
	}
	newRequest, err := RequestAnthropicToOpenAI(r.Request)
	if err != nil {
		return nil, err
	}
	return &LLMRequest{
		Request:   newRequest,
		APIType:   model.APITypeOpenAI,
		Model:     r.Model,
		Stream:    r.Stream,
		RawObject: r.RawObject,
	}, nil
}
