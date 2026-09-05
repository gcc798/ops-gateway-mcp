import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import type { Params } from '../types/api';

export function useListLocation(base: string) {
  const [search, setSearch] = useSearchParams();
  const { id = '' } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const params: Params = { page: '1', page_size: '20', ...Object.fromEntries(search) };
  return {
    params,
    setParams: (values: Params) => setSearch(values),
    selected: id,
    setSelected: (value: string) =>
      navigate(base + (value ? '/' + encodeURIComponent(value) : '') + location.search),
  };
}
