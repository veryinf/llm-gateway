CREATE TABLE IF NOT EXISTS request_logs (
    trace_id          VARCHAR PRIMARY KEY,
    user_id           BIGINT,
    api_key_id        BIGINT,
    user_model        VARCHAR NOT NULL DEFAULT '',
    provider_model    VARCHAR NOT NULL DEFAULT '',
    response_model    VARCHAR NOT NULL DEFAULT '',
    user_api_type     VARCHAR NOT NULL DEFAULT '',
    provider_api_type VARCHAR NOT NULL DEFAULT '',
    passthrough_level VARCHAR NOT NULL DEFAULT 'none',
    summary           VARCHAR NOT NULL DEFAULT '',
    is_stream         BOOLEAN,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    reasoning_tokens  INTEGER DEFAULT 0,
    total_tokens      INTEGER,
    cached_tokens     INTEGER DEFAULT 0,
    is_detail         BOOLEAN DEFAULT FALSE,
    status_code       INTEGER,
    error_message     VARCHAR NOT NULL DEFAULT '',
    duration          BIGINT,
    ip_address        VARCHAR NOT NULL DEFAULT '',
    user_agent        VARCHAR NOT NULL DEFAULT '',
    created_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS request_details (
    trace_id      VARCHAR PRIMARY KEY,
    request       VARCHAR NOT NULL DEFAULT '',
    request_raw   VARCHAR NOT NULL DEFAULT '',
    response      VARCHAR NOT NULL DEFAULT '',
    response_raw  VARCHAR NOT NULL DEFAULT '',
    reasoning     VARCHAR NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS request_chunks (
    chunk_id    BIGINT,
    trace_id    VARCHAR NOT NULL DEFAULT '',
    index       INTEGER,
    type        VARCHAR NOT NULL DEFAULT 'message',
    data        VARCHAR NOT NULL DEFAULT '',
    created_at  TIMESTAMP
);
