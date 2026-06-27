CREATE TABLE IF NOT EXISTS request_logs (
    trace_id          VARCHAR PRIMARY KEY,
    user_id           BIGINT,
    api_key_id        BIGINT,
    user_model        VARCHAR DEFAULT '',
    provider_model    VARCHAR DEFAULT '',
    response_model    VARCHAR DEFAULT '',
    user_api_type     VARCHAR DEFAULT '',
    provider_api_type VARCHAR DEFAULT '',
    passthrough_level VARCHAR DEFAULT 'none',
    summary           VARCHAR DEFAULT '',
    is_stream         BOOLEAN,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    reasoning_tokens  INTEGER DEFAULT 0,
    total_tokens      INTEGER,
    cached_tokens     INTEGER DEFAULT 0,
    is_detail         BOOLEAN DEFAULT FALSE,
    status_code       INTEGER,
    error_message     VARCHAR,
    duration          BIGINT,
    ip_address        VARCHAR DEFAULT '',
    user_agent        VARCHAR DEFAULT '',
    created_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS request_details (
    trace_id      VARCHAR PRIMARY KEY,
    request       VARCHAR DEFAULT '',
    request_raw   VARCHAR DEFAULT '',
    response      VARCHAR DEFAULT '',
    response_raw  VARCHAR DEFAULT '',
    reasoning     VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS request_chunks (
    chunk_id    BIGINT,
    trace_id    VARCHAR,
    index       INTEGER,
    type        VARCHAR DEFAULT 'message',
    data        VARCHAR,
    created_at  TIMESTAMP
);
