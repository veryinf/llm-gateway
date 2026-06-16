import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useMemo, useState } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { FormFieldInput, FormFieldSelect, FormFieldSwitch, FormFieldTextarea } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { modelService, type Model } from '@/services/model';
import { providerService, type Provider } from '@/services/provider';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { useQuery } from '@tanstack/react-query';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { request } from '@/lib';
import type { API } from '@/typings';

export const Route = createFileRoute('/provider-models')({
  component: ProviderModelsPage,
});

const pageInformation: PageInformation = {
  name: 'provider-models',
  entityName: '模型',
  page: { title: '服务商模型', description: '管理各 Provider 下的模型详细信息' },
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
  const [providerFilter, setProviderFilter] = useState<string>('all');

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  const { data: providers } = useQuery({
    queryKey: ['providers-list'],
    queryFn: () => providerService.search({}),
  });

  const providerOptions = (providers?.dataSet ?? []).map((p: Provider) => ({
    label: p.title,
    value: String(p.providerId),
  }));

  const dynamicPageName = useMemo(() => {
    return providerFilter === 'all' ? 'provider-models' : `provider-models-p${providerFilter}`;
  }, [providerFilter]);

  const filteredService = useMemo<API.Service<Model>>(
    () => ({
      ...modelService,
      async search(params) {
        const filters = [...(params.filters ?? [])];
        if (providerFilter !== 'all') {
          filters.push({ field: 'providerId', value: Number(providerFilter) });
        }
        const res = await request.post<API.DataSet<Model>>('/provider-models/search', {
          ...params,
          filters,
        });
        return res.data;
      },
    }),
    [providerFilter],
  );

  const dynamicPageInfo = useMemo(
    () => ({ ...pageInformation, name: dynamicPageName }),
    [dynamicPageName],
  );

  const columns: ColumnDef<Model, any>[] = [
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
      accessorKey: 'provider',
      header: 'Provider',
      meta: { label: 'Provider', className: 'w-[140px]' },
      cell: ({ row }) => row.original.provider?.title ?? '-',
    },
    {
      accessorKey: 'apiType',
      header: 'API 类型',
      meta: { label: 'API 类型', className: 'w-[90px]' },
      cell: ({ row }) => (
        <Badge variant="outline">{row.original.apiType === 'anthropic' ? 'Anthropic' : 'OpenAI'}</Badge>
      ),
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

  const formInitialValue = (_type: string, entity?: Model) => ({
    modelId: entity?.modelId ?? 0,
    providerId: entity?.providerId ?? 0,
    name: entity?.name ?? '',
    apiType: entity?.apiType ?? 'openai',
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

  const apiTypeOptions = [
    { label: 'OpenAI', value: 'openai' },
    { label: 'Anthropic', value: 'anthropic' },
  ];

  const renderForm = (form: any, _entity?: Model) => (
    <div className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto pr-2">
      <div className="text-sm font-medium text-muted-foreground">基础信息</div>
      <FormFieldInput form={form} name="name" title="模型名称" required placeholder="例如: gpt-4o, claude-3-opus" />
      <FormFieldInput form={form} name="displayName" title="展示名" placeholder="用户友好的显示名称" />
      <FormFieldSelect form={form} name="providerId" title="Provider" options={providerOptions} required />
      <FormFieldSelect form={form} name="apiType" title="API 类型" options={apiTypeOptions} required />
      <FormFieldTextarea form={form} name="description" title="描述" placeholder="模型描述信息" rows={2} />

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">容量与限制</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="maxContextTokens" title="最大上下文 (tokens)" type="number" placeholder="128000" />
        <FormFieldInput form={form} name="maxOutputTokens" title="最大输出 (tokens)" type="number" placeholder="4096" />
      </div>

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">速率限制</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="tpm" title="TPM (0=不限)" type="number" tips="Tokens Per Minute" />
        <FormFieldInput form={form} name="qpm" title="QPM (0=不限)" type="number" tips="Queries Per Minute" />
      </div>

      <div className="text-sm font-medium text-muted-foreground border-t pt-4">定价 (per 1M tokens)</div>
      <div className="grid grid-cols-2 gap-4">
        <FormFieldInput form={form} name="inputPrice" title="输入单价 ($)" type="number" placeholder="0.00" />
        <FormFieldInput form={form} name="outputPrice" title="输出单价 ($)" type="number" placeholder="0.00" />
      </div>

      <FormFieldSwitch form={form} name="isActive" title="启用" switchLabel="启用此服务商模型" />
    </div>
  );

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 px-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Provider 筛选:</span>
            <Select value={providerFilter} onValueChange={setProviderFilter}>
              <SelectTrigger className="w-[200px] h-8">
                <SelectValue placeholder="全部 Provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部 Provider</SelectItem>
                {providerOptions.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>
      <Page<Model>
        infomation={dynamicPageInfo}
        columns={columns}
        service={filteredService}
        options={{ showSelectColumn: false }}
        formInitialValue={formInitialValue}
        renderViewAdd={(form) => renderForm(form)}
        renderViewUpdate={(form, entity) => renderForm(form, entity)}
      />
    </div>
  );
}
