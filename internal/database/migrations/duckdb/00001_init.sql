CREATE TABLE IF NOT EXISTS request_logs (
    trace_id          VARCHAR PRIMARY KEY,
    user_id           BIGINT,
    api_key_id        BIGINT,
    model_name        VARCHAR,
    summary           VARCHAR DEFAULT '',
    is_stream         BOOLEAN,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    total_tokens      INTEGER,
    is_detail         BOOLEAN DEFAULT FALSE,
    status_code       INTEGER,
    error_message     VARCHAR,
    latency_ms        BIGINT,
    cost              DOUBLE,
    ip_address        VARCHAR DEFAULT '',
    user_agent        VARCHAR DEFAULT '',
    created_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS request_details (
    trace_id      VARCHAR PRIMARY KEY,
    request_body  VARCHAR DEFAULT '',
    response_body VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS request_chunks (
    chunk_id    BIGINT,
    trace_id    VARCHAR,
    index       INTEGER,
    data        VARCHAR,
    created_at  TIMESTAMP
);
