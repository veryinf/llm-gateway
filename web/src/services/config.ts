import { request } from '@/lib';
import type { API } from '@/typings';

export interface Config {
  id: number;
  key: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export const configService = {
  async list() {
    const res = await request.get<API.DataSet<Config>>('/admin/configs');
    return res.data;
  },

  async update(key: string, value: string) {
    const res = await request.put<API.Data<Config>>('/admin/configs', { key, value });
    return res.data;
  },
};
