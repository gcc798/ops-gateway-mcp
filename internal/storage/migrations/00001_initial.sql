-- +goose Up
CREATE TABLE database_resources (
    name TEXT PRIMARY KEY,
    environment TEXT NOT NULL DEFAULT '',
    driver TEXT NOT NULL,
    dsn TEXT NOT NULL DEFAULT ''
);
CREATE TABLE linux_resources (
    name TEXT PRIMARY KEY,
    environment TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL,
    user TEXT NOT NULL,
    password TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL DEFAULT '',
    known_hosts_path TEXT NOT NULL DEFAULT ''
);
CREATE TABLE kubernetes_resources (
    name TEXT PRIMARY KEY,
    environment TEXT NOT NULL DEFAULT '',
    kubeconfig TEXT NOT NULL,
    context TEXT NOT NULL DEFAULT '',
    token TEXT NOT NULL DEFAULT ''
);
CREATE TABLE operations (
    operation_id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    client TEXT, tool TEXT, environment TEXT, resource_type TEXT, resource TEXT,
    action TEXT, target TEXT, risk TEXT, policy_decision TEXT, policy_reason TEXT,
    status TEXT, duration_ms INTEGER, affected_rows INTEGER,
    statement_hash TEXT, error TEXT,
    resource_revision TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    confirmed_by TEXT NOT NULL DEFAULT ''
);
CREATE TABLE pending_operations (
    operation_id TEXT PRIMARY KEY,
    resource TEXT NOT NULL,
    dialect TEXT NOT NULL,
    statement TEXT NOT NULL,
    statement_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    FOREIGN KEY(operation_id) REFERENCES operations(operation_id)
);
CREATE TABLE pending_actions (
    operation_id TEXT PRIMARY KEY,
    resource_type TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    FOREIGN KEY(operation_id) REFERENCES operations(operation_id)
);
CREATE TABLE api_tokens (
    token_hash TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    scope TEXT NOT NULL,
    expires_at TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL,
    last_used_at TEXT
);
