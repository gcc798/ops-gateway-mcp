-- Run after the gateway has started once so the tables exist.
-- Secrets stay outside SQLite: use environment variable names and local paths.
INSERT INTO database_resources(name, environment, driver, dsn_env) VALUES
  ('dev-postgres', 'dev', 'postgres', 'AI_OPS_GATEWAY_DEV_PG_DSN'),
  ('dev-mysql', 'dev', 'mysql', 'AI_OPS_GATEWAY_DEV_MYSQL_DSN');

INSERT INTO linux_resources(name, environment, address, user, password_env, private_key_path, known_hosts_path) VALUES
  ('dev-linux', 'dev', '192.0.2.10:22', 'ops', '', '~/.ssh/id_ed25519', '~/.ssh/known_hosts');

INSERT INTO kubernetes_resources(name, environment, kubeconfig, context) VALUES
  ('dev-k8s', 'dev', '~/.kube/config', 'dev');
