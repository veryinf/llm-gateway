package model

// ConfigKey 数据库配置键统一定义
const (
	// ConfigKeyLogRetention 日志文件保留天数，值为整数字符串，默认 7
	ConfigKeyLogRetention = "system.log.retention"

	// ConfigKeyRouterPassthrough 透传级别配置
	// 值为 "none" 时禁用透传（默认）
	// 值为 "user" 时启用一级透传：跳过 UserModel，直接匹配 ProviderModel
	// 值为 "provider" 时启用两级透传：跳过 ProviderModel，直接使用 default Provider
	ConfigKeyRouterPassthrough = "system.router.passthrough"

	// ConfigKeyRequestLogDetail 是否记录完整请求/响应 body，值为 "true" 或 "false"，默认 "false"
	ConfigKeyRequestLogDetail = "system.request.log_detail"

	// ConfigKeyRequestRetentionDays 请求日志保留天数，值为整数字符串，默认 90
	ConfigKeyRequestRetentionDays = "system.request.retention_days"
)

type Config struct {
	Key         string `json:"key" gorm:"primaryKey;size:128"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
