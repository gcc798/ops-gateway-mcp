import { useEffect, useState } from 'react';
import { get, message } from '../lib/api';
import type { List, Params } from '../types/api';

export function useList<T>(path: string, token: string, params: Params) {
  const [data, setData] = useState<List<T>>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);
  const query = new URLSearchParams(params).toString();
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    setData(undefined);
    get<List<T>>(path + '?' + query, token, controller.signal)
      .then((value) => {
        if (!controller.signal.aborted) setData(value);
      })
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [path, token, query, revision]);
  return { data, loading, error, refresh: () => setRevision((value) => value + 1) };
}
