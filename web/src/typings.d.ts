import '@tanstack/react-table';
import type { OptionsItem } from './lib';
import type { UseQueryResult } from '@tanstack/react-query';

declare namespace API {
  type PrimaryKeyType = number;
  export type Service<T> = {
    primaryKey: (entity: T) => PrimaryKeyType;
    title(entity: T): string;
    search: (params: SearchParams) => Promise<API.DataSet<T>>;
    fetch: (id: PrimaryKeyType) => Promise<API.Data<T>>;
    add: (params: T) => Promise<API.ResponseStruct>;
    update: (id: PrimaryKeyType, params: Partial<T>) => Promise<API.ResponseStruct>;
    delete: (id: PrimaryKeyType) => Promise<API.ResponseStruct>;
    bulkCreate?: (params: T[]) => Promise<API.ResponseStruct>;
    bulkUpdate?: (params: T[]) => Promise<API.ResponseStruct>;
    bulkDelete?: (ids: PrimaryKeyType[]) => Promise<API.ResponseStruct>;
  };

  export interface SearchParams {
    kw?: string;
    filters?: { field: string, value: unknown; }[];
    pagination?: { index: number, size: number; };
  }

  export interface ResponseStruct {
    errCode: number;
    errMsg: string;
  }

  export interface DataSet<T> {
    errCode: number;
    errMsg: string;
    dataSet: T[];
    total: number;
  }

  export interface Data<T> {
    errCode: number;
    errMsg: string;
    data?: T;
  }
}

declare module '@tanstack/react-table' {
  interface ColumnMeta<TData extends RowData, TValue> {
    /** 列标题 */
    label?: string;
    /** 当前列是否点击展示详情 */
    viewDetail?: boolean;
    /** 列样式，统一设置 */
    className?: string;
    /** 表头样式 */
    thClassName?: string;
    /** 单元格样式 */
    tdClassName?: string;
    /** 列枚举 */
    emuns?: OptionsItem[];
    /** 过滤器变体 */
    variant?: string;
    /** 占位符 */
    placeholder?: string;
    /** 单位 */
    unit?: string;
    /** 范围 */
    range?: [number, number];
    /** 选项 */
    options?: OptionsItem[];
  }
}
