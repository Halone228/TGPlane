CREATE TABLE messages (
    id          BIGSERIAL    PRIMARY KEY,
    session_id  VARCHAR(128) NOT NULL,
    worker_id   VARCHAR(128) NOT NULL DEFAULT '',
    type        VARCHAR(128) NOT NULL,
    payload     JSONB        NOT NULL DEFAULT '{}',
    received_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX messages_session_id_idx ON messages (session_id);
CREATE INDEX messages_received_at_idx ON messages (received_at DESC);
