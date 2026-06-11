import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { providerService, type Provider } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

export const Route = createFileRoute('/providers')({
  component: ProvidersPage,
});

const typeOptions = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Azure', value: 'azure' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'OpenAI Compatible', value: 'openai-compatible' },
  { label: 'Ollama', value: 'ollama' },
];

const pageInformation: PageInformation = {
  name: 'providers',
  entityName: 'Provider',
  page: { title: 'Provider 管理', description: '管理 LLM 服务商配置' },
  breadcrumbs: [{ title: '管理' }, { title: 'Provider' }],
};

const columns: ColumnDef<Provider, any>[] = [
  {
    accessorKey: 'name',
    header: '名称',
    meta: { label: '名称', viewDetail: true },
  },
  {
    accessorKey: 'type',
    header: '类型',
    meta: { label: '类型' },
    cell: ({ row }) => <Badge variant="outline">{typeOptions.find((t) => t.value === row.original.type)?.label ?? row.original.type}</Badge>,
  },
  {
    accessorKey: 'base_url',
    header: 'Base URL',
    meta: { label: 'Base URL' },
  },
  {
    accessorKey: 'priority',
    header: '优先级',
    meta: { label: '优先级' },
  },
  {
    accessorKey: 'is_active',
    header: '状态',
    meta: { label: '状态' },
    cell: ({ row }) => (
      <Badge variant={row.original.is_active ? 'default' : 'destructive'}>
        {row.original.is_active ? '启用' : '禁用'}
      </Badge>
    ),
  },
];

function ProvidersPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  return (
    <Page<Provider>
      infomation={pageInformation}
      columns={columns}
      service={providerService}
      options={{ showSelectColumn: false }}
      formInitialValue={(_type, entity) => ({
        id: 0,
        name: entity?.name ?? '',
        type: entity?.type ?? 'openai',
        base_url: entity?.base_url ?? '',
        api_key: '',
        is_active: entity?.is_active ?? true,
        priority: entity?.priority ?? 0,
        rate_limit_qpm: entity?.rate_limit_qpm ?? 0,
        rate_limit_burst: entity?.rate_limit_burst ?? 0,
        created_at: '',
        updated_at: '',
      })}
      renderViewAdd={(form) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="名称" required placeholder="请输入 Provider 名称" />
          <FormFieldSelect form={form} name="type" title="类型" options={typeOptions} required />
          <FormFieldInput form={form} name="base_url" title="Base URL" required placeholder="https://api.openai.com" />
          <FormFieldInput form={form} name="api_key" title="API Key" placeholder="sk-..." type="password" />
          <FormFieldInput form={form} name="priority" title="优先级" type="number" />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此 Provider" />
        </div>
      )}
      renderViewUpdate={(form, _entity) => (
        <div className="flex flex-col gap-4">
          <FormFieldInput form={form} name="name" title="名称" required />
          <FormFieldSelect form={form} name="type" title="类型" options={typeOptions} required />
          <FormFieldInput form={form} name="base_url" title="Base URL" required />
          <FormFieldInput form={form} name="api_key" title="API Key (留空不修改)" placeholder="留空不修改" type="password" />
          <FormFieldInput form={form} name="priority" title="优先级" type="number" />
          <FormFieldInput form={form} name="rate_limit_qpm" title="QPM 限流 (0=不限)" type="number" />
          <FormFieldInput form={form} name="rate_limit_burst" title="并发上限 (0=默认)" type="number" />
          <FormFieldSwitch form={form} name="is_active" title="启用" switchLabel="启用此 Provider" />
        </div>
      )}
    />
  );
}
