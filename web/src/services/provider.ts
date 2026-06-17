import { request, type OptionsItem } from '@/lib';
import type { API } from '@/typings';
import { useQuery } from '@tanstack/react-query';

export interface Provider {
  providerId: number;
  title: string;
  baseUrl: string;
  apiKey?: string;
  supportOpenai: boolean;
  openaiBaseUrl: string;
  supportAnthropic: boolean;
  anthropicBaseUrl: string;
  preferredApi: string;
  isActive: boolean;
  modelCount?: number;
}

export function useAllProviders() {
  const { data: allProviders = [], ...rest } = useQuery<Provider[]>({
    queryKey: ['all-providers'],
    queryFn: async () => {
      const result = await providerService.search({ pagination: { pageIndex: 1, pageSize: 10000 } });
      return result.dataSet ?? [];
    },
  });
  const allProviderOptions: OptionsItem[] = allProviders.map(p => ({
    label: p.title, value: p.providerId
  }));

  return { allProviders, allProviderOptions, ...rest };
}

export const providerService: API.Service<Provider> = {
  primaryKey: (entity) => entity.providerId,
  title: (entity) => entity.title,

  async search(params) {
    const res = await request.post<API.DataSet<Provider>>('/providers/search', params);
    return res.data;
  },

  async fetch(providerId) {
    const res = await request.post<API.Data<Provider>>('/providers/fetch', { providerId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/providers/add', params);
    return res.data;
  },

  async update(providerId, params) {
    const res = await request.post('/providers/update', { providerId, ...params });
    return res.data;
  },

  async delete(providerId) {
    const res = await request.post('/providers/remove', { providerId });
    return res.data;
  },
};

export async function fetchProviderModels(baseUrl: string, apiKey: string): Promise<{ id: string }[]> {
  const res = await request.post<API.DataSet<{ id: string }>>('/providers/fetch-models', {
    baseUrl,
    apiKey,
  });
  return res.data.dataSet ?? [];
}
