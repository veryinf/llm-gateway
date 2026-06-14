import { request } from '@/lib';
import type { API } from '@/typings';

export interface APIKey {
  id: number;
  user_id: number;
  key: string;
  name: string;
  quota_limit: number;
  quota_used: number;
  rate_limit_qpm: number;
  expires_at: string | null;
  is_active: boolean;
  last_used_at: string | null;
  created_at: string;
}

export interface CreateAPIKeyParams {
  name: string;
  quota_limit?: number;
  rate_limit_qpm?: number;
}

export interface CreateAPIKeyResponse {
  api_key: APIKey;
  raw_key: string;
}

export const apiKeyService = {
  async listByUser(userId: number): Promise<APIKey[]> {
    const res = await request.get<API.DataSet<APIKey>>(`/admin/users/${userId}/api-keys`);
    return res.data.dataSet ?? [];
  },

  async create(userId: number, params: CreateAPIKeyParams): Promise<CreateAPIKeyResponse> {
    const res = await request.post<API.Data<CreateAPIKeyResponse>>(
      `/admin/users/${userId}/api-keys`,
      params,
    );
    return res.data.data!;
  },

  async delete(userId: number, keyId: number): Promise<void> {
    await request.delete(`/admin/users/${userId}/api-keys/${keyId}`);
  },

  async listAll(): Promise<APIKey[]> {
    const res = await request.get<API.DataSet<APIKey>>('/admin/api-keys');
    return res.data.dataSet ?? [];
  },

  async deleteGlobal(keyId: number): Promise<void> {
    await request.delete(`/admin/api-keys/${keyId}`);
  },

  async toggleActive(keyId: number): Promise<{ is_active: boolean }> {
    const res = await request.put<API.Data<{ is_active: boolean }>>(`/admin/api-keys/${keyId}/toggle`);
    return res.data.data!;
  },
};
