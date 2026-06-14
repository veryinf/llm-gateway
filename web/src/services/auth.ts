import { request } from '@/lib';
import type { API } from '@/typings';

export interface SessionUser {
  uid: number;
  username: string;
  name: string;
  phone: string;
  department: string;
  role: 'admin' | 'user' | 'viewer';
  status: string;
}

export interface LoginParams {
  username: string;
  password: string;
}

export async function login(params: LoginParams): Promise<void> {
  const res = await request.post<API.Data<{ token: string }>>('/auth/login', params);
  if (res.data.errCode !== 0) {
    throw new Error(res.data.errMsg || '登录失败');
  }
  localStorage.setItem('accessToken', res.data.data!.token);
}

export async function fetchProfile(): Promise<SessionUser | null> {
  if (!localStorage.getItem('accessToken')) {
    return null;
  }
  try {
    const res = await request.get<API.Data<SessionUser>>('/profile');
    if (res.data.errCode !== 0) {
      return null;
    }
    return res.data.data!;
  } catch {
    return null;
  }
}

export async function logoutApi(): Promise<void> {
  try {
    await request.post('/auth/logout');
  } catch {
    // 即使失败也继续清除本地token
  }
}

export function logoutLocal(): void {
  localStorage.removeItem('accessToken');
}
