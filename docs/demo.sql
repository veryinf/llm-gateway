-- LLM Gateway 测试数据
-- 创建时间: 2026-06-22
-- 注意：此文件仅包含 SQLite 应用数据库的表数据，不包含 DuckDB 的 request_logs 和 request_chunks

-- 1. 用户数据 (users)
INSERT INTO users (username, password, name, phone, department, role, status, access_key, secret_key) VALUES
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '系统管理员', '13800138000', '技术部', 'admin', 'active', 'adminak12345678', 'adminsk1234567890abcdef1234567890abcdef'),
('zhangsan', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '张三', '13800138001', '开发部', 'user', 'active', 'zhangsanak12345', 'zhangsansk1234567890abcdef1234567890abcdef'),
('lisi', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '李四', '13800138002', '产品部', 'user', 'active', 'lisiak123456789', 'lisis234567890abcdef1234567890abcdef'),
('wangwu', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '王五', '13800138003', '测试部', 'viewer', 'active', 'wangwuak1234567', 'wangwusk1234567890abcdef1234567890abcdef'),
('zhaoliu', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '赵六', '13800138004', '运维部', 'user', 'inactive', 'zhaoliuak123456', 'zhaoliusk1234567890abcdef1234567890abcdef');

-- 2. API Keys (user_keys)
INSERT INTO user_keys (uid, key, title, is_active) VALUES
(1, 'sk-admin-key-001', '管理员主密钥', 1),
(1, 'sk-admin-key-002', '管理员备用密钥', 1),
(2, 'sk-zhangsan-key-001', '张三开发密钥', 1),
(2, 'sk-zhangsan-key-002', '张三测试密钥', 0),
(3, 'sk-lisi-key-001', '李四产品密钥', 1),
(4, 'sk-wangwu-key-001', '王五只读密钥', 1),
(5, 'sk-zhaoliu-key-001', '赵六运维密钥', 1);

-- 3. LLM 提供商 (providers)
INSERT INTO providers (title, base_url, api_key, support_open_ai, open_ai_base_url, support_anthropic, anthropic_base_url, preferred_api, is_active, is_default) VALUES
('OpenAI', 'https://api.openai.com', 'sk-openai-api-key-001', 1, 'https://api.openai.com/v1', 0, '', 'openai', 1, 1),
('Anthropic', 'https://api.anthropic.com', 'sk-anthropic-api-key-001', 0, '', 1, 'https://api.anthropic.com', 'anthropic', 1, 0),
('DeepSeek', 'https://api.deepseek.com', 'sk-deepseek-api-key-001', 1, 'https://api.deepseek.com/v1', 0, '', 'openai', 1, 0),
('通义千问', 'https://dashscope.aliyuncs.com', 'sk-qwen-api-key-001', 1, 'https://dashscope.aliyuncs.com/compatible-mode/v1', 0, '', 'openai', 1, 0),
('Kimi', 'https://api.moonshot.cn', 'sk-kimi-api-key-001', 1, 'https://api.moonshot.cn/v1', 0, '', 'openai', 1, 0),
('Ollama', 'http://localhost:11434', '', 1, 'http://localhost:11434/v1', 0, '', 'openai', 1, 0);

-- 4. 提供商模型 (provider_models)
-- OpenAI 模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(1, 'gpt-4o', 'GPT-4o', 'OpenAI 最新旗舰模型', 128000, 4096, 0.0025, 0.01, 100000, 500, 1),
(1, 'gpt-4o-mini', 'GPT-4o Mini', '轻量级 GPT-4o 模型', 128000, 4096, 0.00015, 0.0006, 200000, 1000, 1),
(1, 'gpt-4-turbo', 'GPT-4 Turbo', 'GPT-4 Turbo 模型', 128000, 4096, 0.01, 0.03, 80000, 400, 1),
(1, 'gpt-3.5-turbo', 'GPT-3.5 Turbo', 'GPT-3.5 Turbo 模型', 16385, 4096, 0.0005, 0.0015, 300000, 2000, 1);

-- Anthropic 模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(2, 'claude-3.5-sonnet', 'Claude 3.5 Sonnet', 'Anthropic 最新模型', 200000, 4096, 0.003, 0.015, 80000, 400, 1),
(2, 'claude-3-opus', 'Claude 3 Opus', 'Claude 3 Opus 模型', 200000, 4096, 0.015, 0.075, 50000, 200, 1),
(2, 'claude-3-haiku', 'Claude 3 Haiku', 'Claude 3 Haiku 模型', 200000, 4096, 0.00025, 0.00125, 150000, 800, 1);

-- DeepSeek 模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(3, 'deepseek-chat', 'DeepSeek Chat', 'DeepSeek 对话模型', 32768, 4096, 0.0001, 0.0002, 100000, 500, 1),
(3, 'deepseek-coder', 'DeepSeek Coder', 'DeepSeek 代码模型', 32768, 4096, 0.0001, 0.0002, 100000, 500, 1);

-- 通义千问模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(4, 'qwen-turbo', '通义千问 Turbo', '通义千问快速模型', 8192, 2048, 0.0003, 0.0006, 200000, 1000, 1),
(4, 'qwen-plus', '通义千问 Plus', '通义千问增强模型', 32768, 8192, 0.004, 0.012, 100000, 500, 1),
(4, 'qwen-max', '通义千问 Max', '通义千问旗舰模型', 32768, 8192, 0.02, 0.06, 50000, 200, 1);

-- Kimi 模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(5, 'moonshot-v1-8k', 'Kimi 8K', 'Kimi 8K 上下文模型', 8192, 4096, 0.0012, 0.0012, 100000, 500, 1),
(5, 'moonshot-v1-32k', 'Kimi 32K', 'Kimi 32K 上下文模型', 32768, 4096, 0.024, 0.024, 50000, 200, 1),
(5, 'moonshot-v1-128k', 'Kimi 128K', 'Kimi 128K 上下文模型', 131072, 4096, 0.06, 0.06, 20000, 100, 1);

-- Ollama 模型
INSERT INTO provider_models (provider_id, name, display_name, description, max_context_tokens, max_output_tokens, input_price, output_price, tpm, qpm, is_active) VALUES
(6, 'llama3', 'Llama 3', 'Meta Llama 3 模型', 8192, 2048, 0, 0, 50000, 200, 1),
(6, 'qwen2', 'Qwen 2', '阿里云 Qwen 2 模型', 32768, 8192, 0, 0, 50000, 200, 1);

-- 5. 用户模型 (user_models)
INSERT INTO user_models (name, display_name, description, is_active) VALUES
('gpt-4o', 'GPT-4o', '通用 GPT-4o 模型', 1),
('gpt-4o-mini', 'GPT-4o Mini', '轻量级 GPT-4o 模型', 1),
('claude-3.5-sonnet', 'Claude 3.5 Sonnet', 'Claude 3.5 Sonnet 模型', 1),
('deepseek-chat', 'DeepSeek Chat', 'DeepSeek 对话模型', 1),
('qwen-turbo', '通义千问 Turbo', '通义千问快速模型', 1),
('moonshot-v1-8k', 'Kimi 8K', 'Kimi 8K 模型', 1),
('llama3', 'Llama 3', '本地 Llama 3 模型', 1);

-- 6. 配置项 (configs)
INSERT INTO configs (key, value, description) VALUES
('system.log.retention', '7', '日志文件保留天数'),
('system.router.passthrough', 'none', '透传级别配置'),
('system.max_tokens_per_request', '4096', '单次请求最大 token 数'),
('system.rate_limit_enabled', 'true', '是否启用速率限制'),
('system.cost_tracking_enabled', 'true', '是否启用费用追踪');