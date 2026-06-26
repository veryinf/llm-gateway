package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"llm-gateway/internal/model"

	"github.com/tidwall/gjson"
)

// LLMResponse LLM 响应封装
type LLMResponse struct {
	APIType    model.LLMAPIType // API 类型：openai / anthropic
	Converted  bool             //是否经过转换
	Response   *http.Response   //响应数据(经过转换)
	StatusCode int              // HTTP 状态码
	RawObject  *gjson.Result    // 原始响应体解析结果
}

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
		RawObject:  &result,
	}, nil
}

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
