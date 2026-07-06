CREATE TABLE IF NOT EXISTS stats_hourly (
    hour            TIMESTAMP,
    user_id         BIGINT,
    user_model      VARCHAR,
    provider_model  VARCHAR,
    request_count   INTEGER DEFAULT 0,
    success_count   INTEGER DEFAULT 0,
    error_count     INTEGER DEFAULT 0,
    prompt_tokens   BIGINT DEFAULT 0,
    completion_tokens BIGINT DEFAULT 0,
    reasoning_tokens BIGINT DEFAULT 0,
    total_tokens    BIGINT DEFAULT 0,
    total_duration  BIGINT DEFAULT 0,
    UNIQUE (hour, user_id, user_model, provider_model)
);