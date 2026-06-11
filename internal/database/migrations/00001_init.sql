-- +goose Up

CREATE TABLE IF NOT EXISTS request_logs (
    trace_id          VARCHAR(36),
    user_id           BIGINT,
    api_key_id        BIGINT,
    provider_id       BIGINT,
    model_name        VARCHAR(128),
    is_stream         BOOLEAN,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    total_tokens      INTEGER,
    request_body      TEXT DEFAULT '',
    response_body     TEXT DEFAULT '',
    is_detail         BOOLEAN DEFAULT FALSE,
    status_code       INTEGER,
    error_message     VARCHAR(1024),
    latency_ms        BIGINT,
    cost              DOUBLE,
    ip_address        VARCHAR(64) DEFAULT '',
    user_agent        VARCHAR(512) DEFAULT '',
    created_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS request_chunks (
    id            BIGINT,
    trace_id      VARCHAR(36),
    chunk_index   INTEGER,
    chunk_data    TEXT,
    created_at    TIMESTAMP
);

-- +goose Down

DROP TABLE IF EXISTS request_chunks;
DROP TABLE IF EXISTS request_logs;
