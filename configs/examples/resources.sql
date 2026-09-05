-- 首次启动网关创建资源表后，再执行本示例。
-- 内部开发模式允许明文凭据，请在本地替换以下示例值。
INSERT INTO database_resources(name, environment, driver, dsn) VALUES
  ('dev-postgres', 'dev', 'postgres', 'postgres://ops:example-password@127.0.0.1:5432/app?sslmode=disable'),
  ('dev-mysql', 'dev', 'mysql', 'ops:example-password@tcp(127.0.0.1:3306)/app');

INSERT INTO linux_resources(name, environment, address, user, password, private_key, known_hosts_path) VALUES
  ('dev-linux', 'dev', '192.0.2.10:22', 'ops', 'example-password', '', '~/.ssh/known_hosts');

INSERT INTO kubernetes_resources(name, environment, kubeconfig, context, token) VALUES
  ('dev-k8s', 'dev', '~/.kube/config', 'dev', '');
