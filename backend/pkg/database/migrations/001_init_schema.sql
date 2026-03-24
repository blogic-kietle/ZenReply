-- ZenReply Database Schema
-- Migration: 001_init_schema
-- Architecture: User Token (xoxp-) only — no Bot Token column.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── users ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slack_user_id VARCHAR(64)  NOT NULL UNIQUE,
    slack_team_id VARCHAR(64)  NOT NULL,
    slack_name    VARCHAR(255) NOT NULL DEFAULT '',
    email         VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url    TEXT         NOT NULL DEFAULT '',
    -- access_token stores the Slack User Token (xoxp-...), encrypted at rest.
    access_token  TEXT         NOT NULL DEFAULT '',
    token_scope   TEXT         NOT NULL DEFAULT '',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_slack_user_id ON users(slack_user_id);
CREATE INDEX IF NOT EXISTS idx_users_slack_team_id ON users(slack_team_id);

-- ── user_settings ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_settings (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id            UUID         NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    default_message    TEXT         NOT NULL DEFAULT 'I am currently in a deep work session and will reply as soon as I am available.',
    default_reason     VARCHAR(255) NOT NULL DEFAULT 'Deep Work',
    cooldown_minutes   INT          NOT NULL DEFAULT 3,
    -- whitelist / blacklist stored as JSON arrays of Slack user IDs
    whitelist          JSONB        NOT NULL DEFAULT '[]',
    blacklist          JSONB        NOT NULL DEFAULT '[]',
    reply_in_thread    BOOLEAN      NOT NULL DEFAULT TRUE,
    notify_on_resume   BOOLEAN      NOT NULL DEFAULT FALSE,
    auto_reply_enabled BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- ── deep_work_sessions ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS deep_work_sessions (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason     VARCHAR(500) NOT NULL DEFAULT '',
    start_time TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    end_time   TIMESTAMPTZ,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id    ON deep_work_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_is_active  ON deep_work_sessions(is_active);
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON deep_work_sessions(start_time DESC);

-- ── message_logs ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS message_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id      UUID        REFERENCES deep_work_sessions(id) ON DELETE SET NULL,
    sender_slack_id VARCHAR(64) NOT NULL,
    channel_id      VARCHAR(64) NOT NULL,
    original_ts     VARCHAR(32) NOT NULL DEFAULT '',
    thread_ts       VARCHAR(32) NOT NULL DEFAULT '',
    message_text    TEXT        NOT NULL DEFAULT '',
    auto_reply_text TEXT        NOT NULL DEFAULT '',
    was_sent        BOOLEAN     NOT NULL DEFAULT FALSE,
    error_message   TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_logs_user_id    ON message_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_session_id ON message_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_created_at ON message_logs(created_at DESC);

-- ── updated_at trigger ────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_sessions_updated_at
    BEFORE UPDATE ON deep_work_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
