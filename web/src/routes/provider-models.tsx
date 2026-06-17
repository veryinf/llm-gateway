import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { providerModelService, type ProviderModel } from '@/services/provider-model';
import { useAllProviders } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

export const Route = createFileRoute('/provider-models')({
  component: ProviderModelsPage,
});

const pageInformation: PageInformation = {
  name: 'provider-models',
  entityName: '模型',
  page: { title: '服务商模型', description: '管理各服务商下的模型详细信息' },
  breadcrumbs: [{ title: '上游' }, { title: '服务商模型' }],
};

function formatTokens(value: number) {
  if (!value) return '-';
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}K`;
  return String(value);
}

function formatPrice(value: number) {
  if (!value) return '-';
  return `$${value.toFixed(2)}`;
}

function ProviderModelsPage() {
  const { setBreadcrumbs } = useBreadcrumb();
  const { allProviderOptions, isLoading } = useAllProviders();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const columns: ColumnDef<ProviderModel, any>[] = [
    {
      accessorKey: 'name',
      header: '模型名称',
      meta: { label: '模型名称', className: 'w-[180px]', viewDetail: true },
    },
    {
      accessorKey: 'displayName',
      header: '展示名',
      meta: { label: '展示名', className: 'w-[140px]' },
      cell: ({ row }) => row.original.displayName || '-',
    },
    {
      accessorKey: 'providerId',
      header: '服务商',
      enableColumnFilter: true,
      meta: { label: '服务商', className: 'w-[140px]', emuns: allProviderOptions },
    },
    {
      accessorKey: 'maxContextTokens',
      header: '上下文',
      meta: { label: '上下文', className: 'w-[90px]' },
      cell: ({ row }) => formatTokens(row.original.maxContextTokens),
    },
    {
      accessorKey: 'maxOutputTokens',
      header: '最大输出',
      meta: { label: '最大输出', className: 'w-[90px]' },
      cell: ({ row }) => formatTokens(row.original.maxOutputTokens),
    },
    {
      accessorKey: 'tpm',
      header: 'TPM',
      meta: { label: 'TPM', className: 'w-[80px]' },
      cell: ({ row }) => (row.original.tpm ? formatTokens(row.original.tpm) : '-'),
    },
    {
      accessorKey: 'qpm',
      header: 'QPM',
      meta: { label: 'QPM', className: 'w-[80px]' },
      cell: ({ row }) => (row.original.qpm ? String(row.original.qpm) : '-'),
    },
    {
      accessorKey: 'inputPrice',
      header: '输入单价',
      meta: { label: '输入单价', className: 'w-[90px]' },
      cell: ({ row }) => formatPrice(row.original.inputPrice),
    },
    {
      accessorKey: 'outputPrice',
      header: '输出单价',
      meta: { label: '输出单价', className: 'w-[90px]' },
      cell: ({ row }) => formatPrice(row.original.outputPrice),
    },
    {
      accessorKey: 'isActive',
      header: '状态',
      meta: { label: '状态', className: 'w-[70px]' },
      cell: ({ row }) => (
        <Badge variant={row.original.isActive ? 'default' : 'destructive'}>
          {row.original.isActive ? '启用' : '禁用'}
        </Badge>
      ),
    },
  ];

  const formInitialValue = (_type: string, entity?: ProviderModel) => ({
    modelId: entity?.modelId ?? 0,
    providerId: String(entity?.providerId ?? 0) as any,
    name: entity?.name ?? '',
    displayName: entity?.displayName ?? '',
    description: entity?.description ?? '',
    maxContextTokens: entity?.maxContextTokens ?? 0,
    maxOutputTokens: entity?.maxOutputTokens ?? 0,
    inputPrice: entity?.inputPrice ?? 0,
    outputPrice: entity?.outputPrice ?? 0,
    tpm: entity?.tpm ?? 0,
    qpm: entity?.qpm ?? 0,
    isActive: entity?.isActive ?? true,
  });

  return (
    <Page<ProviderModel>
      infomation={pageInformation}
      ready={!isLoading}
      columns={columns}
      service={providerModelService}
      options={{ showSelectColumn: false }}
      formInitialValue={formInitialValue}
      renderViewDetail={(entity) => <ProviderModelDetail entity={entity} />}
      renderViewForm={(form, _entity, _formType) => {
        return <div className="flex flex-col gap-4 overflow-y-auto pr-2">
          <div className="text-sm font-medium text-muted-foreground">基础信息</div>
          <div className="grid grid-cols-3 gap-4">
            <FormFieldInput className='flex-1' form={form} name="name" title="模型名称" required placeholder="例如: gpt-4o, claude-3-opus" />
            <FormFieldInput className='flex-1' form={form} name="displayName" title="展示名" placeholder="用户友好的显示名称" />
            <FormFieldSelect className='flex-1' form={form} name="providerId" title="服务商" options={allProviderOptions} required />
          </div>
          <FormFieldTextarea form={form} name="description" title="描述" placeholder="模型描述信息" rows={2} />
          <FormFieldSwitch form={form} name="isActive" title="启用" switchLabel="启用此服务商模型" />

          <div className="text-sm font-medium text-muted-foreground border-t pt-4">更多信息</div>
          <div className="grid grid-cols-3 gap-4">
            <FormFieldInput form={form} name="maxContextTokens" title="最大上下文 (tokens)" type="number" placeholder="128000" />
            <FormFieldInput form={form} name="maxOutputTokens" title="最大输出 (tokens)" type="number" placeholder="4096" />
            <FormFieldInput form={form} name="tpm" title="TPM (0=不限)" type="number" tips="Tokens Per Minute" />
            <FormFieldInput form={form} name="qpm" title="QPM (0=不限)" type="number" tips="Queries Per Minute" />
            <FormFieldInput form={form} name="inputPrice" title="输入单价 ($)" type="number" placeholder="0.00" />
            <FormFieldInput form={form} name="outputPrice" title="输出单价 ($)" type="number" placeholder="0.00" />
          </div>
        </div>;
      }}
    />
  );
}

function ProviderModelDetail({ entity }: { entity: ProviderModel; }) {
  return (
    <Descriptions
      title="模型信息"
      labelClassName="w-24"
      items={[
        { label: '模型名称', value: <span className="font-mono text-xs">{entity.name}</span> },
        { label: '展示名', value: entity.displayName || '-' },
        { label: '服务商', value: entity.provider?.title ?? '-' },
        { label: '描述', value: entity.description || '-' },
        { label: '最大上下文', value: formatTokens(entity.maxContextTokens) },
        { label: '最大输出', value: formatTokens(entity.maxOutputTokens) },
        { label: 'TPM', value: entity.tpm ? formatTokens(entity.tpm) : '-' },
        { label: 'QPM', value: entity.qpm ? String(entity.qpm) : '-' },
        { label: '输入单价', value: formatPrice(entity.inputPrice) },
        { label: '输出单价', value: formatPrice(entity.outputPrice) },
        {
          label: '状态',
          value: <Badge variant={entity.isActive ? 'default' : 'destructive'}>{entity.isActive ? '启用' : '禁用'}</Badge>,
        },
      ]}
    />
  );
}
