import { useEffect, useState, useCallback } from 'react';
import { message } from '../lib/api';

const tokenKey = 'ops-gateway-mcp-token';

// Hook 将登录状态与 sessionStorage 绑定，刷新页面后可恢复会话。
export function useAuth() {
  const [token, setToken] = useState(() => sessionStorage.getItem(tokenKey) || '');
  const [verified, setVerified] = useState(false);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const logout = useCallback(() => {
    sessionStorage.removeItem(tokenKey);
    setToken('');
    setVerified(false);
    setError('');
  }, []);
  useEffect(() => {
    window.addEventListener('gateway-unauthorized', logout);
    return () => window.removeEventListener('gateway-unauthorized', logout);
  }, [logout]);
  useEffect(() => {
    if (!token) return;
    const controller = new AbortController();
    setError('');
    fetch('/auth/login', {
      method: 'POST',
      headers: { Authorization: 'Bearer ' + token },
      signal: controller.signal,
    })
      .then((response) => {
        if (controller.signal.aborted) return;
        if (response.status === 401) logout();
        else if (response.ok) setVerified(true);
        else throw new Error(response.status + ' ' + response.statusText);
      })
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [token, revision, logout]);
  const login = (value: string) => {
    sessionStorage.setItem(tokenKey, value);
    setToken(value);
    setVerified(true);
  };
  return { token, verified, error, login, logout, retry: () => setRevision((value) => value + 1) };
}
