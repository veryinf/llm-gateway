package model

import "time"

type RequestLog struct {
	TraceID          string    `db:"trace_id"          json:"traceId"`
	UserID           uint      `db:"user_id"           json:"userId"`
	APIKeyID         uint      `db:"api_key_id"        json:"apiKeyId"`
	UserModel        string    `db:"user_model"        json:"userModel"`
	ProviderModel    string    `db:"provider_model"    json:"providerModel"`
	UserApiType      string    `db:"user_api_type"     json:"userApiType"`
	ProviderApiType  string    `db:"provider_api_type" json:"providerApiType"`
	PassthroughLevel string    `db:"passthrough_level" json:"passthroughLevel"`
	Summary          string    `db:"summary"           json:"summary"`
	IsStream         bool      `db:"is_stream"         json:"isStream"`
	PromptTokens     int       `db:"prompt_tokens"     json:"promptTokens"`
	CompletionTokens int       `db:"completion_tokens" json:"completionTokens"`
	TotalTokens      int       `db:"total_tokens"      json:"totalTokens"`
	CachedTokens     int       `db:"cached_tokens"     json:"cachedTokens"`
	IsDetail         bool      `db:"is_detail"         json:"isDetail"`
	StatusCode       int       `db:"status_code"       json:"statusCode"`
	ErrorMessage     string    `db:"error_message"     json:"errorMessage"`
	Duration         int64     `db:"duration"          json:"duration"`
	IPAddress        string    `db:"ip_address"        json:"ipAddress"`
	UserAgent        string    `db:"user_agent"        json:"userAgent"`
	CreatedAt        time.Time `db:"created_at"        json:"createdAt"`
}

type RequestDetail struct {
	TraceID      string `db:"trace_id"      json:"traceId"`
	RequestBody  string `db:"request_body"  json:"requestBody"`
	ResponseBody string `db:"response_body" json:"responseBody"`
}

type RequestChunk struct {
	ChunkID   uint      `db:"chunk_id"   json:"chunkId"`
	TraceID   string    `db:"trace_id"   json:"traceId"`
	Index     int       `db:"index"      json:"index"`
	Data      string    `db:"data"       json:"data"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}
