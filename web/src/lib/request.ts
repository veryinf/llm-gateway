import axios, { type InternalAxiosRequestConfig } from 'axios';
import md5 from 'md5';
import { toast } from 'sonner';
import { router } from './root-provider';

// ============================================================
// 鉴权模式配置
// ============================================================
export type AuthMode = 'token' | 'signature';

const AUTH_MODE: AuthMode = (import.meta.env.VITE_AUTH_MODE as AuthMode) || 'token';

export function getAuthMode(): AuthMode {
  return AUTH_MODE;
}

// API 签名模式配置
const API_KEY = import.meta.env.VITE_API_KEY || '';
const API_SECRET = import.meta.env.VITE_API_SECRET || '';

// ============================================================
// 获取鉴权头
// ============================================================
function getAuthHeaders(): Record<string, string> {
  if (AUTH_MODE === 'signature') {
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const signature = md5(timestamp + API_SECRET);
    return {
      'X-Api-Key': API_KEY,
      'X-Api-Time': timestamp,
      'X-Api-Signature': signature,
    };
  }
  const token = localStorage.getItem('accessToken');
  if (token) {
    return { Authorization: `Bearer ${token}` };
  }
  return {};
}

// ============================================================
// 导航守卫：防止 401 重复跳转
// ============================================================
let isNavigatingToLogin = false;

// ============================================================
// Axios 实例
// ============================================================
const request = axios.create({
  baseURL: '/api',
  adapter: 'fetch',
});

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const headers = getAuthHeaders();
    for (const [key, value] of Object.entries(headers)) {
      config.headers[key] = value;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

request.interceptors.response.use(
  (response) => {
    const { data } = response;
    if (data && typeof data.errCode === 'number' && data.errCode !== 0) {
      toast.error('接口错误', { description: data.errMsg || '请求失败', position: 'bottom-right' });
    }
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      if (!isNavigatingToLogin) {
        isNavigatingToLogin = true;
        if (AUTH_MODE === 'token') {
          toast.warning('登录已失效', { description: '请重新登录', position: 'bottom-right' });
          localStorage.removeItem('accessToken');
          router.navigate({ to: '/auth/login' }).finally(() => {
            isNavigatingToLogin = false;
          });
        } else {
          toast.error('鉴权失败', { description: '请检查 AKSK 配置', position: 'bottom-right' });
          isNavigatingToLogin = false;
        }
      }
    } else if (error.response?.status === 403) {
      toast.error('权限不足', { description: '您无法访问此功能', position: 'bottom-right' });
    } else {
      toast.error('网络错误', { description: '请检查网络连接', position: 'bottom-right' });
    }
    return Promise.reject(error);
  },
);

export { request };
