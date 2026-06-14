CREATE TABLE IF NOT EXISTS request_logs (
    trace_id          VARCHAR,
    user_id           BIGINT,
    api_key_id        BIGINT,
    provider_id       BIGINT,
    model_name        VARCHAR,
    is_stream         BOOLEAN,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    total_tokens      INTEGER,
    request_body      VARCHAR DEFAULT '',
    response_body     VARCHAR DEFAULT '',
    is_detail         BOOLEAN DEFAULT FALSE,
    status_code       INTEGER,
    error_message     VARCHAR,
    latency_ms        BIGINT,
    cost              DOUBLE,
    ip_address        VARCHAR DEFAULT '',
    user_agent        VARCHAR DEFAULT '',
    created_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS request_chunks (
    id            BIGINT,
    trace_id      VARCHAR,
    chunk_index   INTEGER,
    chunk_data    VARCHAR,
    created_at    TIMESTAMP
);
