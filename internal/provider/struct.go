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
	APIType model.LLMAPIType // API 类型：openai / anthropic
	Request *http.Request    // 原始 HTTP 请求（用于转发，可能被转换）
	Model   string           // 模型名称（用于路由判断）
	Stream  bool             // 是否流式（用于路由判断）
	BodyRaw *gjson.Result    // 原始请求体字节（用于日志记录）
}

// LLMResponse LLM 响应封装
type LLMResponse struct {
	APIType    model.LLMAPIType // API 类型：openai / anthropic
	Converted  bool             //是否经过转换
	Response   *http.Response   //响应数据(经过转换)
	StatusCode int              // HTTP 状态码
	BodyRaw    *gjson.Result    // 原始响应体解析结果
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"ownedBy"`
}

// ==================== 请求解析函数 ====================

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
		APIType: apiType,
		Request: req,
		Model:   result.Get("model").String(),
		Stream:  result.Get("stream").Bool(),
		BodyRaw: &result,
	}, nil
}

// ==================== 响应构造函数 ====================

// NewLLMResponse 从 *http.Response 构造 LLMResponse
// 读取响应体并保留副本，同时用 gjson 解析
func NewLLMResponse(resp *http.Response, apiType model.LLMAPIType) (*LLMResponse, error) {
	// 使用 TeeReader 读取 body 并同时复制一份
	var bodyBuf bytes.Buffer
	bodyBytes, err := io.ReadAll(io.TeeReader(resp.Body, &bodyBuf))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(&bodyBuf)
	// 使用 gjson 解析必要字段
	if !gjson.ValidBytes(bodyBytes) {
		return nil, fmt.Errorf("invalid response body")
	}
	result := gjson.ParseBytes(bodyBytes)
	return &LLMResponse{
		APIType:    apiType,
		Converted:  false,
		Response:   resp,
		StatusCode: resp.StatusCode,
		BodyRaw:    &result,
	}, nil
}

// ==================== LLMRequest 转换方法 ====================

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
		Request: newRequest,
		APIType: model.APITypeAnthropic,
		Model:   r.Model,
		Stream:  r.Stream,
		BodyRaw: r.BodyRaw,
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
		Request: newRequest,
		APIType: model.APITypeOpenAI,
		Model:   r.Model,
		Stream:  r.Stream,
		BodyRaw: r.BodyRaw,
	}, nil
}

// ==================== LLMResponse 转换方法 ====================

// ToAnthropic 将响应转为 Anthropic 格式
func (r *LLMResponse) ToAnthropic() (*LLMResponse, error) {
	if r.APIType == model.APITypeAnthropic {
		return r, nil
	}
	newBody, err := ResponseOpenAIToAnthropic(r.Response.Body)
	if err != nil {
		return nil, err
	}
	r.Converted = true
	r.Response.Body = newBody
	r.APIType = model.APITypeAnthropic
	return r, nil
}

// ToOpenAI 将响应转为 OpenAI 格式
func (r *LLMResponse) ToOpenAI() (*LLMResponse, error) {
	if r.APIType == model.APITypeOpenAI {
		return r, nil
	}
	newBody, err := ResponseAnthropicToOpenAI(r.Response.Body)
	if err != nil {
		return nil, err
	}
	r.Converted = true
	r.Response.Body = newBody
	r.APIType = model.APITypeOpenAI
	return r, nil
}
