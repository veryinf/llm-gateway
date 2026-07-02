import { createFileRoute } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useStore } from '@tanstack/react-form';
import type { ColumnDef } from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import { FlaskConical, Loader2 } from 'lucide-react';
import { Page, type FormType, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSwitch, FormFieldTagsInput } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Loading } from '@/components/loader';
import { providerService, fetchProviderModels, testProviderModel, type Provider } from '@/services/provider';
import { providerModelService } from '@/services/provider-model';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';
import { toast } from 'sonner';
import type { EasyFormApi } from '@/components/form/utils';

export const Route = createFileRoute('/providers')({
  component: ProvidersPage,
});

const pageInformation: PageInformation = {
  name: 'providers',
  entityName: '服务商',
  page: { title: '服务商管理', description: '管理 LLM 服务商配置' },
  breadcrumbs: [{ title: '上游' }, { title: 'LLM 服务商' }],
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

const columns: ColumnDef<Provider, any>[] = [
  {
    accessorKey: 'title',
    header: '名称',
    meta: { label: '名称', viewDetail: true },
  },
  {
    accessorKey: 'protocol',
    header: '协议',
    meta: { label: '协议', className: 'w-[120px]' },
    cell: ({ row }) => (
      <div className="flex gap-1">
        {row.original.supportOpenai && <Badge variant="secondary">OpenAI</Badge>}
        {row.original.supportAnthropic && <Badge variant="secondary">Anthropic</Badge>}
      </div>
    ),
  },
  {
    accessorKey: 'modelCount',
    header: '模型数',
    meta: { label: '模型数', className: 'w-[80px]' },
    cell: ({ row }) => <Badge variant={(row.original.modelCount ?? 0) > 0 ? 'default' : 'secondary'}>{row.original.modelCount ?? 0}</Badge>,
  },
  {
    accessorKey: 'isDefault',
    header: '默认',
    meta: { label: '默认', className: 'w-[60px]' },
    cell: ({ row }) => row.original.isDefault ? <Badge variant="default">是</Badge> : null,
  },
  {
    accessorKey: 'isActive',
    header: '状态',
    meta: { label: '状态', className: 'w-[70px]' },
    cell: ({ row }) => <Badge variant={row.original.isActive ? 'default' : 'destructive'}>{row.original.isActive ? '启用' : '禁用'}</Badge>,
  },
];

// ---------- ProviderForm (新增/编辑共用) ----------

function ProviderForm({ form, entity, formType }: { form: EasyFormApi<any>; entity?: Provider; formType: FormType; }) {
  const isEdit = formType == 'update';
  console.log(isEdit);

  const v = useStore(form.store ?? form, (s: any) => s.values ?? s) as any;
  const baseUrl = v?.baseUrl ?? '';

  // 编辑模式：加载已有模型
  useQuery({
    enabled: isEdit,
    queryKey: ['provider-models', entity?.providerId],
    queryFn: async () => {
      const result = await providerModelService.search({
        filters: [{ field: 'providerId', value: entity!.providerId }],
      });
      const models = (result.dataSet ?? []).map((m) => m.name);
      form.setFieldValue('models', models);
      return result.dataSet;
    },
  });

  async function handleFetchModels() {
    if (!baseUrl) { toast.error('请先填写 BaseURL'); return []; }
    const apiKey = v?.apiKey || (entity as any)?.rawApiKey || '';
    if (!apiKey) { toast.error('请填写 API Key'); return []; }
    try {
      const fetched = await fetchProviderModels(baseUrl, apiKey);
      toast.success(`获取到 ${fetched.length} 个模型`);
      return fetched.map((m) => m.id);
    } catch { toast.error('获取模型列表失败'); return []; }
  }

  return (
    <div className="grid grid-cols-12 gap-4">
      <FormFieldInput className="col-span-12" form={form} name="title" title="名称" required placeholder="服务商名称" />
      <FormFieldInput className="col-span-12" form={form} name="baseUrl" title="Base URL" required placeholder="https://api.example.com" />
      <FormFieldInput className="col-span-12" form={form} name="apiKey" title="API Key" required={!isEdit} placeholder={isEdit ? '留空不修改' : 'sk-...'} type="password" />
      <div className='col-span-12 grid grid-cols-12 gap-4'>
        <FormFieldSwitch className="col-span-2" form={form} name="supportOpenai" title="OpenAI协议接入" switchLabel="支持" />
        {form.getFieldValue("supportOpenai") && (
          <FormFieldInput className="col-span-10" form={form} name="openaiBaseUrl" title="OpenAI BaseURL" placeholder={baseUrl ? `${baseUrl}/v1` : 'https://api.example.com/v1'} />
        )}
      </div>
      <div className='col-span-12  grid grid-cols-12 gap-4'>
        <FormFieldSwitch className="col-span-2" form={form} name="supportAnthropic" title="Anthropic协议接入" switchLabel="支持" />
        {form.getFieldValue("supportAnthropic") && (
          <FormFieldInput className="col-span-10" form={form} name="anthropicBaseUrl" title="Anthropic BaseURL" placeholder={baseUrl ? `${baseUrl}/anthropic/v1` : 'https://api.example.com/anthropic/v1'} />
        )}
      </div>
      <FormFieldSwitch className="col-span-2" form={form} name="isActive" title="启用当前服务商" switchLabel="启用" />
      <FormFieldSwitch className="col-span-2" form={form} name="isDefault" title="设为默认服务商" switchLabel="默认" />
      <FormFieldTagsInput
        className="col-span-12 border-t pt-4"
        form={form}
        name="models"
        title="上游模型"
        placeholder="输入模型名称后按 Enter 添加..."
        titleMore={
          <Button type="button" size="sm" variant="outline" onClick={async () => {
            const tags = await handleFetchModels();
            if (tags.length > 0) {
              const current: string[] = (form.getFieldValue('models') as string[] | undefined) ?? [];
              const existing = new Set(current);
              form.setFieldValue('models', [...current, ...tags.filter((t) => !existing.has(t))]);
            }
          }}>
            从 API 获取
          </Button>
        }
      />
    </div>
  );
}

// ---------- ProvidersPage ----------

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
      optionColumn={(column, domRender) => ({ ...column, cell: (res) => domRender(res.row.original) })}
      renderViewDetail={(entity) => <ProviderDetail entity={entity} />}
      formInitialValue={(formType, entity) => (formType == 'add' ? {
        supportOpenai: true,
        supportAnthropic: true,
        isActive: true,
        isDefault: false,
        models: [],
      } : { ...entity!, apiKey: '', rawApiKey: entity?.apiKey ?? '' })}
      formAddValidator={(e) => {
        if (!e.supportOpenai && !e.supportAnthropic) { toast.error('请至少支持一种协议'); return false; }
        return true;
      }}
      formUpdateValidator={(e) => {
        if (!e.supportOpenai && !e.supportAnthropic) { toast.error('请至少支持一种协议'); return false; }
        return true;
      }}
      renderViewForm={(form, entity, formType) => <ProviderForm form={form} entity={entity} formType={formType} />}
    />
  );
}

// ---------- ProviderDetail ----------

function ProviderDetail({ entity }: { entity: Provider; }) {
  const pid = entity.providerId;

  const { data: models = [], isLoading: modelsLoading } = useQuery({
    queryKey: ['provider-models', pid],
    queryFn: async () => {
      const result = await providerModelService.search({
        filters: [{ field: 'providerId', value: pid }],
      });
      return result.dataSet ?? [];
    },
  });

  const [testStates, setTestStates] = useState<Record<number, { status: 'testing' | 'success' | 'error'; latencyMs?: number; error?: string; }>>({});
  const [batchTesting, setBatchTesting] = useState(false);

  async function handleTestOne(modelId: number, modelName: string) {
    setTestStates((prev) => ({ ...prev, [modelId]: { status: 'testing' } }));
    try {
      const result = await testProviderModel(pid, modelName);
      setTestStates((prev) => ({
        ...prev,
        [modelId]: {
          status: result.success ? 'success' : 'error',
          latencyMs: result.latencyMs,
          error: result.error,
        },
      }));
    } catch (e: any) {
      setTestStates((prev) => ({
        ...prev,
        [modelId]: { status: 'error', error: e?.message ?? '请求失败' },
      }));
    }
  }

  async function handleBatchTest() {
    if (models.length === 0) return;
    setBatchTesting(true);
    const initial: Record<number, { status: 'testing'; }> = {};
    for (const m of models) initial[m.modelId] = { status: 'testing' };
    setTestStates(initial);

    let successCount = 0;
    let failCount = 0;
    await Promise.allSettled(
      models.map(async (m) => {
        try {
          const result = await testProviderModel(pid, m.name);
          setTestStates((prev) => ({
            ...prev,
            [m.modelId]: {
              status: result.success ? 'success' : 'error',
              latencyMs: result.latencyMs,
              error: result.error,
            },
          }));
          if (result.success) successCount++; else failCount++;
        } catch (e: any) {
          setTestStates((prev) => ({
            ...prev,
            [m.modelId]: { status: 'error', error: e?.message ?? '请求失败' },
          }));
          failCount++;
        }
      })
    );

    setBatchTesting(false);
    toast.success(`批量测试完成：成功 ${successCount}，失败 ${failCount}`);
  }

  return (
    <div className="flex flex-col gap-4">
      <Descriptions
        title="服务商信息"
        labelClassName="w-30"
        items={[
          { label: '名称', value: entity.title },
          { label: '状态', value: <Badge variant={entity.isActive ? 'default' : 'destructive'}>{entity.isActive ? '启用' : '禁用'}</Badge> },
          {
            label: '协议',
            value: (
              <div className="flex gap-1">
                {entity.supportOpenai && <Badge variant="secondary">OpenAI</Badge>}
                {entity.supportAnthropic && <Badge variant="secondary">Anthropic</Badge>}
              </div>
            ),
          },
          { label: '默认服务商', value: entity.isDefault ? <Badge variant="default">是</Badge> : <Badge variant="secondary">否</Badge> },
          { label: 'Base URL', value: <span className="font-mono text-xs">{entity.baseUrl}</span> },
        ]}
      />

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>上游模型 ({models.length})</CardTitle>
            <Button
              size="sm"
              variant="outline"
              onClick={handleBatchTest}
              disabled={batchTesting || modelsLoading || models.length === 0}
            >
              {batchTesting ? (
                <><Loader2 className="animate-spin" /> 测试中...</>
              ) : (
                <><FlaskConical /> 批量测试</>
              )}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {modelsLoading ? <Loading size={20} /> : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>模型名称</TableHead>
                  <TableHead>上下文</TableHead>
                  <TableHead>最大输出</TableHead>
                  <TableHead>输入单价</TableHead>
                  <TableHead>输出单价</TableHead>
                  <TableHead className="text-center">状态</TableHead>
                  <TableHead className="w-40">测试</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-muted-foreground text-center">暂无模型</TableCell></TableRow>
                ) : models.map((m) => {
                  const state = testStates[m.modelId];
                  return (
                    <TableRow key={m.modelId}>
                      <TableCell className="font-mono text-xs">{m.displayName ? `${m.displayName} (${m.name})` : m.name}</TableCell>
                      <TableCell>{formatTokens(m.maxContextTokens)}</TableCell>
                      <TableCell>{formatTokens(m.maxOutputTokens)}</TableCell>
                      <TableCell>{formatPrice(m.inputPrice)}</TableCell>
                      <TableCell>{formatPrice(m.outputPrice)}</TableCell>
                      <TableCell className="text-center"><Badge variant={m.isActive ? 'default' : 'destructive'}>{m.isActive ? '启用' : '禁用'}</Badge></TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleTestOne(m.modelId, m.name)}
                            disabled={state?.status === 'testing' || batchTesting}
                            title="测试模型"
                          >
                            {state?.status === 'testing' ? <Loader2 className="animate-spin" /> : <FlaskConical />}
                          </Button>
                          {state?.status === 'success' && (
                            <Badge variant="default">✓ {state.latencyMs}ms</Badge>
                          )}
                          {state?.status === 'error' && (
                            <Badge variant="destructive" title={state.error}>失败</Badge>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
