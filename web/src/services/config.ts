import { request } from '@/lib';
import type { API } from '@/typings';

// 数据库配置键常量，与后端 internal/model/config.go 保持一致
export const CONFIG_KEYS = {
  LOG_RETENTION: 'system.log.retention',
  ROUTER_PASSTHROUGH: 'system.router.passthrough',
  REQUEST_LOG_DETAIL: 'system.request.log_detail',
  REQUEST_RETENTION_DAYS: 'system.request.retention_days',
} as const;

export const configService = {
  async get(keys: string[]): Promise<Record<string, string>> {
    const res = await request.post<API.Data<Record<string, string>>>('/config/get', { keys });
    return res.data.data!;
  },

  async save(configs: Record<string, string>) {
    const res = await request.post<API.ResponseStruct>('/config/save', { configs });
    return res.data;
  },
};
