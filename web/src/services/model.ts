import { request } from '@/lib';
import type { API } from '@/typings';
import type { Provider } from './provider';

export interface Model {
  modelId: number;
  providerId: number;
  name: string;
  apiType: 'openai' | 'anthropic';
  displayName: string;
  description: string;
  maxContextTokens: number;
  maxOutputTokens: number;
  inputPrice: number;
  outputPrice: number;
  tpm: number;
  qpm: number;
  isActive: boolean;
  provider?: Provider;
}

export const modelService: API.Service<Model> = {
  primaryKey: (entity) => entity.modelId,
  title: (entity) => entity.displayName || entity.name,

  async search(params) {
    const res = await request.post<API.DataSet<Model>>('/provider-models/search', params);
    return res.data;
  },

  async fetch(modelId) {
    const res = await request.post<API.Data<Model>>('/provider-models/fetch', { modelId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/provider-models/add', params);
    return res.data;
  },

  async update(modelId, params) {
    const res = await request.post('/provider-models/update', { modelId, ...params });
    return res.data;
  },

  async delete(modelId) {
    const res = await request.post('/provider-models/remove', { modelId });
    return res.data;
  },
};
