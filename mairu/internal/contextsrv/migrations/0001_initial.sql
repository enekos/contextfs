-- Initial schema for Mairu context server

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    category TEXT NOT NULL,
    owner TEXT NOT NULL,
    importance INT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    moderation_status TEXT NOT NULL,
    moderation_reasons TEXT NOT NULL DEFAULT '[]',
    review_required BOOLEAN NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',
    moderation_status TEXT NOT NULL,
    moderation_reasons TEXT NOT NULL DEFAULT '[]',
    review_required BOOLEAN NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS context_nodes (
    uri TEXT PRIMARY KEY,
    project TEXT NOT NULL DEFAULT '',
    parent_uri TEXT NULL,
    name TEXT NOT NULL,
    abstract TEXT NOT NULL,
    overview TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',
    moderation_status TEXT NOT NULL,
    moderation_reasons TEXT NOT NULL DEFAULT '[]',
    review_required BOOLEAN NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS moderation_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    project TEXT NOT NULL DEFAULT '',
    decision TEXT NOT NULL,
    reasons TEXT NOT NULL DEFAULT '[]',
    review_status TEXT NOT NULL DEFAULT 'pending',
    reviewer_decision TEXT NOT NULL DEFAULT '',
    reviewer TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    policy_version TEXT NOT NULL DEFAULT 'v1',
    review_required BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL DEFAULT 'system',
    details TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_outbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    op_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    payload_hash TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bash_history (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL DEFAULT '',
    command TEXT NOT NULL,
    exit_code INT NOT NULL,
    duration_ms INT NOT NULL,
    output TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_memories_project_created ON memories(project, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_skills_project_created ON skills(project, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_project_parent ON context_nodes(project, parent_uri);
CREATE INDEX IF NOT EXISTS idx_moderation_pending ON moderation_events(review_status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON search_outbox(status, next_attempt_at, id);
CREATE INDEX IF NOT EXISTS idx_bash_history_project_created ON bash_history(project, created_at DESC);
