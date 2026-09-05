export const message = (error: unknown) => (error instanceof Error ? error.message : String(error));

// 集中处理请求、鉴权和错误，页面组件只关心业务数据。
export async function request(path: string, token: string, options: RequestInit = {}) {
  const response = await fetch(path, {
    ...options,
    cache: 'no-store',
    headers: { Authorization: 'Bearer ' + token, ...options.headers },
  });
  if (response.status === 401 && path !== '/auth/login')
    window.dispatchEvent(new Event('gateway-unauthorized'));
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string } | null;
    throw new Error(body?.message || response.status + ' ' + response.statusText);
  }
  return response;
}
export async function get<T>(path: string, token: string, signal?: AbortSignal): Promise<T> {
  return (await request(path, token, { signal })).json() as Promise<T>;
}
