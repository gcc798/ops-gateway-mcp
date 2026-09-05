-- +goose Up
CREATE INDEX IF NOT EXISTS database_resources_environment ON database_resources(environment, name);
CREATE INDEX IF NOT EXISTS linux_resources_environment ON linux_resources(environment, name);
CREATE INDEX IF NOT EXISTS kubernetes_resources_environment ON kubernetes_resources(environment, name);
CREATE INDEX IF NOT EXISTS operations_time ON operations(julianday(timestamp) DESC, operation_id DESC);
CREATE INDEX IF NOT EXISTS operations_tool_time ON operations(tool, julianday(timestamp) DESC);
CREATE INDEX IF NOT EXISTS operations_type_time ON operations(resource_type, julianday(timestamp) DESC);
