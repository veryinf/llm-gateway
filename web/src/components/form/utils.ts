import type { FieldApi, FieldOptions, FormOptions, ReactFormExtendedApi } from '@tanstack/react-form';

export type EasyFormOptions<T> = FormOptions<T, any, any, any, any, any, any, any, any, any, any, any>;
export type EasyFormApi<T> = ReactFormExtendedApi<T, any, any, any, any, any, any, any, any, any, any, any>;
export type EasyFormMeta<TEntity, TState> = { original?: TEntity; state?: TState };
export type EasyFieldApi<T> = FieldApi<T, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any, any>;
export type EasyFieldOptions<T> = FieldOptions<T, any, any, any, any, any, any, any, any, any, any, any>;
